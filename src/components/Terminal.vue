<template>
  <div class="console" id="terminal"></div>
</template>

<script>
  import {Terminal} from "xterm"
  import {FitAddon} from "xterm-addon-fit"
  import "xterm/css/xterm.css"
  import LocalEchoController from "../assets/js/local-echo/LocalEchoController"

  export default {
    name: "Terminal",
    data() {
      return {
        term: null,
        fitter: null,
      }
    },
    methods: {
      initTerm() {
        let terminalContainer = document.getElementById("terminal")
        this.term = new Terminal({
          cursorBlink: true,
          fontSize: 20,
          fontFamily: "'Roboto Mono', monospace",

        })
        const fitAddon = new FitAddon()
        this.term.loadAddon(fitAddon)
        this.term.open(terminalContainer)
        fitAddon.fit()
        this.fitter = fitAddon
        // Create a local echo controller
        const localEcho = new LocalEchoController(this.term)
        // Create some auto-completion handlers
        localEcho.addAutocompleteHandler((index) => {
          if (index !== 0) return []
          return ["cat", "ls", "clear"]
        })
        localEcho.addAutocompleteHandler((index) => {
          if (index !== 1) return []
          return ["resume.md", "exec.sh"]
        })

        const help = "dunno lol"
        // Infinite loop of reading lines
        const readLine = () => {
          localEcho.read(`guest@${window.location.hostname}~$ `).then((input) => {
            let filteredInput = input.split(" ").filter(item => {
              return item !== ""
            })
            if (filteredInput.length === 0) { // just "enter"
              readLine()
            } else if (filteredInput[0] === "clear") {
              this.term.clear()
              readLine()
            } else if (filteredInput[0] === "ls") {
              localEcho.println("resume.md exec.sh")
              readLine()
            } else if (filteredInput[0] === "cat" && filteredInput[1] === "resume.md") {
              let resume = `Hi, i'm Matt.\nSo you found my playground, good for you!`
              localEcho.println(resume)
              readLine()
            } else {
              localEcho.println(help)
              readLine()
            }
          })
        }
        readLine()
      },
      onResize() {
        if (this.term !== null) {
          this.fitter.fit()
        }
      },
    },
    mounted() {
      setTimeout(() => {
        this.initTerm()
      }, 100)
      window.addEventListener('resize', this.onResize)
    },
    beforeDestroy() {
      if (this.term !== null) {
        this.term.dispose()
      }
      window.removeEventListener('resize', this.onResize)
    },
  }
</script>

<style scoped>
  .console {
    position: absolute;
  }

</style>