<script>
// @ts-nocheck
  //import {HandleError} from '../wailsjs/go/main/App.js'

  window.displayError = (msg) => {
    console.log(msg)
  }

  class Message {
    constructor(sender, content, sentByMe) {
      this.sender = sender
      this.content = content
      this.sentByMe = sentByMe
    }
  }
  var messages = []
  messages[0] = new Message("x123", "hello bigga", false)
  messages[1] = new Message("balls", "hello ziga", true)

  var inputs = {} 
  inputs.ipDomainInput = ""
</script>

<main>
  <div id="control-panel-master">
    <div id="domain-panel">
      <div>
        server ip / domain
        <input id="domain-input" bind:value={inputs.ipDomainInput}>
      </div>
      <button>
        &gt;
      </button>
    </div>
    <div id="info-panel">
    </div>
    <div id="request-panel">
    </div>
  </div>

  {#each messages as message}
    {#if !message.sentByMe}
      <div class="msg">
        {message.sender}: {message.content}
      </div> 
    {:else}
      <div class="msg sent-by-me">
        {message.content} :{message.sender}
      </div>
    {/if}
  {/each}
       

  <!--<div id="title-app-name">inviv v2</div>-->
</main>
