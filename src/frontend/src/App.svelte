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

    let messages: Message[] = []
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

    let connected = false
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
</script>

<main>
    {#if connected}
    <div id="status-bar">
        <div class="field">
            <button on:click={() => copyToClipboard(joinedGroupKey)}>group key:</button>
            <input disabled style="filter: none;" bind:value={joinedGroupKey}>
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