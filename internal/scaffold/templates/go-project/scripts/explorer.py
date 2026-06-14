#!/usr/bin/env python3
"""Explore directory tree and return structured JSON."""
import argparse, fnmatch, json, os


def run(args):
    path = args.path
    if not os.path.isdir(path):
        return {"error": f"Directory not found: {path}"}

    result = {"files": [], "dirs": 0, "total_files": 0, "languages": {}}
    for root, dirs, files in os.walk(path):
        depth = root.replace(path, "").count(os.sep)
        if depth > args.depth:
            continue
        for f in files:
            if fnmatch.fnmatch(f, args.pattern):
                ext = os.path.splitext(f)[1]
                result["languages"][ext] = result["languages"].get(ext, 0) + 1
                result["files"].append(
                    {
                        "path": os.path.join(root, f),
                        "type": ext,
                        "size": os.path.getsize(os.path.join(root, f)),
                    }
                )
        result["dirs"] += len(dirs)
    result["total_files"] = len(result["files"])
    return result


def main():
    p = argparse.ArgumentParser(description="Explore directory tree")
    p.add_argument("--path", required=True)
    p.add_argument("--pattern", default="*")
    p.add_argument("--depth", type=int, default=3)
    args = p.parse_args()
    result = run(args)
    print(json.dumps(result))


if __name__ == "__main__":
    main()
