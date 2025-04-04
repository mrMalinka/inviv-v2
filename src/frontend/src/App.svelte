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

    let joinedGroupKey: string = ""
    EventsOn("key-update", (key: string) => {
        joinedGroupKey = key
    })

    let messages: Message[] = []
    let messageLadder: HTMLDivElement;
    EventsOn("new-message", (sender: string, contents: string, byMe: boolean) => {
        const shouldScroll = messageLadder.scrollTop + messageLadder.clientHeight >= messageLadder.scrollHeight - 10;

        messages = [...messages, new Message(sender, contents, byMe)];

        setTimeout(() => {
            if (shouldScroll) {
                messageLadder.scrollTop = messageLadder.scrollHeight;
            }
        }, 0);
    });

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

    function inputKeydown(event) {
        if (event.key == "Enter") {
            SendTextMessage(inputs.message)
            inputs.message = ""
        }
    }
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

        <div id="message-ladder" bind:this={messageLadder}>
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

        <input bind:value={inputs.message} id="input-main" on:keydown={inputKeydown}>
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