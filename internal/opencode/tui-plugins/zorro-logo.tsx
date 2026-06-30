// @ts-nocheck
/** @jsxImportSource @opentui/solid */
import type { TuiPlugin, TuiThemeCurrent } from "@opencode-ai/plugin/tui"
import { useTerminalDimensions } from "@opentui/solid"
import { createMemo } from "solid-js"

/**
 * Zorro Hacker — Logo para el dashboard de OpenCode.
 * Se muestra en el slot home_logo.
 * Responsive: versión completa o compacta según tamaño de terminal.
 */

const id = "zorro-logo"

const zorroArt = [
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                   *     -:                                 ",
  "                                  #@@*.  @@%:                               ",
  "                                 .%*=@% *@-#@*                              ",
  "                                 -@*=*@%%#=**@#                             ",
  "                                 @@-++%@@:+**#@=                            ",
  "                                #@+*%%*%++#@%*@%                            ",
  "                               +@@*#*=#++%@@@*%@+                           ",
  "                             =@#==++=++++#%*%%**%#+                         ",
  "                            #@-==++==+++++++++++*#%@-                       ",
  "                           .%#=+==##++++++++++++***%@-                      ",
  "                           #@#-+%+#*++++=+=+=++++*%%%@-                     ",
  "                        .#@#-+++++++++++-:=++-=+++*@@@#                     ",
  "                      =@@*+=++=-:..:.   :   : -=++*#@-                      ",
  "                     %@*@#-. ..--:::-+=::.    :+****@*                      ",
  "                      +@@#+*%+::.*@@@#:::.    =+*#%*@=                      ",
  "                         .:+*%@%%%:@@:::.   .++**@@@@.                      ",
  "                               .*-@#:.    :++**#@%.-+                       ",
  "                                =@+:.   -++***%%+                           ",
  "                               :@#:.  .++**#@%*                             ",
  "                               -@*:  -+*%@%*.                               ",
  "                               :%*: =*%@#.                                  ",
  "                                =@+:%@#                                     ",
  "                                 -@@@:                                      ",
  "                                   *.                                       ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
  "                                                                             ",
]

const compactArt = ["  🦊 ZyroCLI — Zorro Naranja  "]

const Logo = (props: { theme: TuiThemeCurrent }) => {
  const dim = useTerminalDimensions()
  const lines = createMemo(() => {
    const term = dim()
    return term.height >= zorroArt.length + 6 && term.width >= 68
      ? zorroArt
      : compactArt
  })

  return (
    <box flexDirection="column" alignItems="center">
      {lines().map((line) => (
        <text fg={props.theme.accent}>{line}</text>
      ))}
    </box>
  )
}

const tui: TuiPlugin = async (api) => {
  api.slots.register({
    id,
    order: 100,
    slots: {
      home_logo(ctx) {
        return <Logo theme={ctx.theme.current} />
      },
    },
  })
}

const plugin = { id: "zorro-logo", tui }
export default plugin
