<template>
  <div id="app">
    <h1 class="title"><span class="pink">$</span> E.XEC.SH</h1>
    <Terminal></Terminal>
  </div>
</template>

<script>
import Terminal from "./components/Terminal";

export default {
  name: 'app',
  components: {
    Terminal,
  }
}
</script>

<style>
#app {
  font-family: Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
}
  html {
    background-color: black;
  }

  .title {
    color: white;
    font-size: 49px;
  }

  .pink {
    color: deeppink;
  }
</style>
