<script>
// @ts-nocheck
  import {Connect, SendMessage} from '../wailsjs/go/main/App.js';
  import * as runtime from '../wailsjs/runtime/runtime.js'; 

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

  function receiveMessage(sender, contents) {
    console.log(sender)
    console.log(contents)
    messages = [...messages, new Message(sender, contents, false)]
  }
  runtime.EventsOn("msg", receiveMessage)

  var inputs = {} 
  inputs.msgMain = ""
  inputs.ipDomainInput = ""
  inputs.newGCWhenConnecting = true
</script>

<main>
  <div id="control-panel-master">
    <div id="domain-panel">
      <div>
        server ip / domain
        <input id="domain-input" bind:value={inputs.ipDomainInput} >
        <button on:click={() => inputs.newGCWhenConnecting = !inputs.newGCWhenConnecting}>
          {inputs.newGCWhenConnecting ? "make new" : "join"}
        </button>
      </div>
      <button on:click={() => Connect(inputs.ipDomainInput, inputs.newGCWhenConnecting)}>
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
  <input bind:value={inputs.msgMain}>
  <button on:click={() => SendMessage(inputs.msgMain)}>send</button>
       

  <!--<div id="title-app-name">inviv v2</div>-->
</main>
