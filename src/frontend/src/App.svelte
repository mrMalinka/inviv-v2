<script lang="ts">
    import {Connect, SendTextMessage} from "../wailsjs/go/main/App"
    import {EventsOn} from "../wailsjs/runtime/runtime"

    class Message {
        constructor(
            public sender: string,
            public contents: string,
            public byMe: boolean
        ) {}
    }

    // TODO: fix names, structure this a lil
    let joinedGroupKey: string = ""
    EventsOn("key-update", (key: string) => {
        joinedGroupKey = key
    })

    let messages: Message[] = []
    EventsOn("new-message", (sender: string, contents: string, byMe: boolean) => {
        messages = [...messages, new Message(sender, contents, byMe)]
    })

    let inputs: {
        groupKey: string
        makeNew: boolean
        message: string
    } = {
        groupKey: "",
        makeNew: false,
        message: "",
    }
</script>

<main>
    <div id="control-panel">
        <input bind:value={inputs.groupKey} />
        <button on:click={() => inputs.makeNew = !inputs.makeNew}>
            {inputs.makeNew ? "new" : "join"}
        </button>
        <button on:click={() => Connect(inputs.makeNew, inputs.groupKey)}>
            connect
        </button>
    </div>

    {joinedGroupKey}
    {#each messages as msg}
        <p class:byme={msg.byMe}>{msg.sender}: {msg.contents}</p>
    {/each}
    <input bind:value={inputs.message} />
    <button on:click={() => SendTextMessage(inputs.message)}>
        send
    </button>
</main>