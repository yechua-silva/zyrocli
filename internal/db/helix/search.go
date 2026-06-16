package helix

import (
	"context"
	"math"
	"sort"
	"sync"

	helixsdk "github.com/helixdb/helix-db/sdks/go"
)

// hybridDefaultK es el valor por defecto de k para RRF.
const hybridDefaultK = 60

// HybridSearch ejecuta búsqueda vectorial + BM25 en paralelo y fusiona resultados
// usando Reciprocal Rank Fusion (RRF). Si embedding está vacío, solo ejecuta BM25.
// Si ambas búsquedas fallan, retorna el primer error. Si una falla, retorna la otra.
func HybridSearch(ctx context.Context, client *SDKClient, query string, embedding []float32, opts HybridSearchOptions) ([]SearchResultRow, error) {
	if opts.MaxResults <= 0 {
		opts.MaxResults = 10
	}
	if opts.MaxResults > 100 {
		opts.MaxResults = 100
	}

	type partialResult struct {
		results []SearchResultRow
		err     error
	}

	vectorCh := make(chan partialResult, 1)
	textCh := make(chan partialResult, 1)

	var wg sync.WaitGroup

	// Vector search (solo si hay embedding)
	if len(embedding) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := vectorSearch(ctx, client, embedding, opts)
			vectorCh <- partialResult{results, err}
		}()
	} else {
		vectorCh <- partialResult{nil, nil}
	}

	// Text BM25 search
	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := textBM25Search(ctx, client, query, opts)
		textCh <- partialResult{results, err}
	}()

	wg.Wait()
	close(vectorCh)
	close(textCh)

	vResult := <-vectorCh
	tResult := <-textCh

	if vResult.err != nil && tResult.err != nil {
		return nil, vResult.err
	}
	if vResult.err != nil {
		return tResult.results, nil
	}
	if tResult.err != nil {
		return vResult.results, nil
	}

	// Fusionar con RRF
	return fuseRRF(vResult.results, tResult.results, hybridDefaultK, opts.MaxResults), nil
}

// vectorSearch ejecuta búsqueda vectorial contra HelixDB.
func vectorSearch(ctx context.Context, client *SDKClient, embedding []float32, opts HybridSearchOptions) ([]SearchResultRow, error) {
	label := resolveSearchLabel(opts.NodeLabels)

	q := helixsdk.ReadQuery("vector_search").
		VarAs("results",
			helixsdk.G().VectorSearchNodes(label, "content", embedding, opts.MaxResults).ValueMap(),
		).
		Returning("results")

	var raw map[string]interface{}
	if err := client.Exec(ctx, q, &raw); err != nil {
		return nil, err
	}

	return parseSearchResults(raw, "vector", opts.MinScore), nil
}

// textBM25Search ejecuta búsqueda de texto BM25 contra HelixDB.
func textBM25Search(ctx context.Context, client *SDKClient, query string, opts HybridSearchOptions) ([]SearchResultRow, error) {
	label := resolveSearchLabel(opts.NodeLabels)

	q := helixsdk.ReadQuery("text_search").
		VarAs("results",
			helixsdk.G().TextSearchNodes(label, "content", query, opts.MaxResults).ValueMap(),
		).
		Returning("results")

	var raw map[string]interface{}
	if err := client.Exec(ctx, q, &raw); err != nil {
		return nil, err
	}

	return parseSearchResults(raw, "text", opts.MinScore), nil
}

// resolveSearchLabel determina el label a buscar. Si no hay labels definidos,
// usa "Fact" como label por defecto.
func resolveSearchLabel(labels []string) string {
	if len(labels) > 0 {
		return labels[0]
	}
	return "Fact"
}

// parseSearchResults extrae resultados del mapa de respuesta de HelixDB.
// El formato esperado es {"results": [{"$id": ..., "content": ..., "score": ...}, ...]}.
func parseSearchResults(raw map[string]interface{}, source string, minScore float64) []SearchResultRow {
	var results []SearchResultRow
	if data, ok := raw["results"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				r := SearchResultRow{
					Source: source,
				}
				if id, ok := m["$id"].(float64); ok {
					r.ID = int(id)
				}
				if content, ok := m["content"].(string); ok {
					r.Content = content
				}
				if label, ok := m["label"].(string); ok {
					r.Label = label
				}
				if score, ok := m["score"].(float64); ok {
					r.Score = score
				}
				if r.Score >= minScore {
					results = append(results, r)
				}
			}
		}
	}
	return results
}

// ---------------------------------------------------------------------------
// RRF Fusion (T-2.7)
// ---------------------------------------------------------------------------

// fuseRRF fusiona resultados de búsqueda vectorial y textual usando
// Reciprocal Rank Fusion. Normaliza scores a [0,1] por lista antes de
// combinar. Deduplica por ID.
func fuseRRF(vector, text []SearchResultRow, k, maxResults int) []SearchResultRow {
	if len(vector) == 0 {
		return text
	}
	if len(text) == 0 {
		return vector
	}

	// Normalizar scores dentro de cada lista a [0,1]
	normalizeScores(vector)
	normalizeScores(text)

	type rankedItem struct {
		row   SearchResultRow
		score float64
	}

	rankMap := make(map[int]*rankedItem)

	// Calcular RRF score para cada resultado
	for i, r := range vector {
		rank := i + 1
		score := 1.0 / float64(k+rank)
		rankMap[r.ID] = &rankedItem{row: r, score: score}
	}

	for i, r := range text {
		rank := i + 1
		score := 1.0 / float64(k+rank)
		if existing, ok := rankMap[r.ID]; ok {
			existing.score += score
			if r.Score > existing.row.Score {
				existing.row.Score = r.Score
			}
		} else {
			rankMap[r.ID] = &rankedItem{row: r, score: score}
		}
	}

	// Convertir a slice y ordenar por score descendente
	items := make([]*rankedItem, 0, len(rankMap))
	for _, item := range rankMap {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	// Limitar a maxResults
	if len(items) > maxResults {
		items = items[:maxResults]
	}

	result := make([]SearchResultRow, len(items))
	for i, item := range items {
		result[i] = item.row
		result[i].Score = math.Round(item.score*10000) / 10000
	}

	return result
}

// normalizeScores normaliza scores a [0,1] dividiendo por el máximo.
func normalizeScores(results []SearchResultRow) {
	if len(results) == 0 {
		return
	}
	maxScore := results[0].Score
	for _, r := range results {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}
	if maxScore > 0 {
		for i := range results {
			results[i].Score = results[i].Score / maxScore
		}
	}
}
