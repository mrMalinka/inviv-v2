<script>
  //import {ConnectToServer} from '../wailsjs/go/main/App.js'

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

</script>

<main>
  <div id="control-panel">
    <div id="title-app-name">inviv v2</div>
  </div>
  <div id="chat-box">
    {#each messages as message}
      <div class="msg" class:sent-by-me={message.sentByMe}>
        {message.content}
      </div> 
    {/each}
  </div>
</main>
