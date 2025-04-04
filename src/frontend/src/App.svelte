<script lang="ts">
    import { Connect, SendTextMessage } from "../wailsjs/go/main/App"
    import { EventsOn } from "../wailsjs/runtime/runtime"

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

    let messages: Message[] = [
        new Message("498577a3-e35c-45a3-8aef-07e62468b7c9", "hello world", false),
        new Message("498577a3-e35c-45a3-8aef-07e62468b7c9", "asdjidsaijsdajidass huisdfahuiasdf asdfhu8fasdhuafsd asfdhu8ifasdhuiasdfhui asasdfhuiasdfhiu", true)
    ]
    EventsOn("new-message", (sender: string, contents: string, byMe: boolean) => {
        messages = [...messages, new Message(sender, contents, byMe)]
    })

    let inputs: {
        groupKey: string
        makeNew: boolean
        message: string
        domain: string
    } = {
        domain: "",
        groupKey: "",
        makeNew: false,
        message: "",
    }

    let connected = true
    EventsOn("connection-change", (conn: boolean) => {
        connected = conn
    })

    async function copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    }
    joinedGroupKey = "44c3fbd4-b861-4a81-9101-ed129a5ffd87"
</script>

<main>
    {#if connected}
    
    <div id="main-ui">
        <div id="status-bar">
            <div class="field">
                <button on:click={() => copyToClipboard(joinedGroupKey)}>group key:</button>
                <input disabled style="filter: none;" bind:value={joinedGroupKey}>
            </div>
        </div>

        <div id="message-ladder">
            {#each messages as msg}
                <div class="ladder-step">
                    <div class={msg.byMe ? "message byme" : "message"}>
                        <div class="message-top">
                            {msg.byMe ? "you" : msg.sender}
                        </div>

                        <div class="message-contents">
                            {msg.contents}
                        </div>
                    </div> 
                </div>
            {/each}
        </div>
    </div>

    {:else}

    <div id="connect-page">
        <div class="field">
            <div>server</div>
            <input bind:value={inputs.domain}>
        </div>

        <div class="field">
            <button
                style={inputs.makeNew ? "filter: blur(1px);" : ""}
                on:click={() => {
                    inputs.makeNew = !inputs.makeNew
                    inputs.groupKey = ""
                }}
            >key</button>

            <input disabled={inputs.makeNew} bind:value={inputs.groupKey}>
        </div>

        <button id="connect-button" on:click={() => {
            Connect(inputs.domain, inputs.makeNew, inputs.groupKey)
        }}>connect</button>
    </div>

    {/if}
</main>