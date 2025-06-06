<script lang="ts">
	import {
		Connect,
		Disconnect,
		SendTextMessage,
	} from "../wailsjs/go/main/App";
	import { EventsOn } from "../wailsjs/runtime/runtime";

	export class Message {
		constructor(
			public sender: string,
			public contents: string,
			public type: string,
		) {
			const validTypes: string[] = ["info", "me", "them"];
			if (!validTypes.includes(type)) {
				throw new Error(`invalid message type: ${type}`);
			}
		}
	}

	// group key

	let joinedGroupKey: string = "";
	EventsOn("key-update", (key: string) => {
		joinedGroupKey = key;
	});

	// main messages

	let messages: Message[] = [];
	let messageLadder: HTMLDivElement;
	EventsOn(
		"new-message",
		(sender: string, contents: string, type: string) => {
			const shouldScroll =
				messageLadder.scrollTop + messageLadder.clientHeight >=
				messageLadder.scrollHeight - 10;
			messages = [...messages, new Message(sender, contents, type)];
			setTimeout(() => {
				if (shouldScroll) {
					messageLadder.scrollTop = messageLadder.scrollHeight;
				}
			}, 0);
		},
	);

	// inputting

	let inputs = {
		groupKey: "",
		makeNew: false,
		message: "",
		domain: "",
	};

	// connection handling

	let connected = false;
	EventsOn("connection-change", (conn: boolean) => {
		connected = conn;
		messages = [];
	});

	// helper

	function inputKeydown(event: { key: string }) {
		if (event.key !== "Enter" || inputs.message.length === 0) {
			return;
		}
		if (inputs.message === "/leave") {
			Disconnect();
			inputs.message = "";
			return;
		}
		SendTextMessage(inputs.message);
		inputs.message = "";
	}
	async function copyToClipboard(text: string) {
		try {
			await navigator.clipboard.writeText(text);
		} catch (err) {
			console.error("Failed to copy:", err);
		}
	}
</script>

<main>
	{#if connected}
		<div class="chat-container">
			<div class="top-bar">
				<div class="copy-input-wrapper">
					<input
						class="readonly-input noblur"
						disabled={true}
						bind:value={joinedGroupKey}
						aria-label="group key"
					/>
					<button
						class="copy-btn"
						on:click={() => copyToClipboard(joinedGroupKey)}
						aria-label="copy group key"
					>
						<svg
							viewBox="0 0 24 24"
							fill="none"
							style="margin: 3px; border-radius: 6px;"
						>
							<g id="SVGRepo_bgCarrier" stroke-width="0"></g>
							<g
								id="SVGRepo_tracerCarrier"
								stroke-linecap="round"
								stroke-linejoin="round"
							></g>
							<!-- FIXME: icon is not colored -->
							<g id="SVGRepo_iconCarrier">
								<path
									d="M17.5 14H19C20.1046 14 21 13.1046 21 12V5C21 3.89543 20.1046 3 19 3H12C10.8954 3 10 3.89543 10 5V6.5M5 10H12C13.1046 10 14 10.8954 14 12V19C14 20.1046 13.1046 21 12 21H5C3.89543 21 3 20.1046 3 19V12C3 10.8954 3.89543 10 5 10Z"
									stroke="#0e0e0e"
									stroke-width="1.5"
									stroke-linecap="round"
									stroke-linejoin="round"
								></path>
							</g>
						</svg>
					</button>
				</div>
			</div>

			<div class="messages" bind:this={messageLadder}>
				{#each messages as msg}
					{#if msg.type === "me" || msg.type == "them"}
						<div
							class="message-row {msg.type === 'me'
								? 'me'
								: 'them'}"
						>
							<div class="message-bubble">
								<div class="sender">
									{msg.type === "me" ? "You" : msg.sender}
								</div>
								<div class="text">{msg.contents}</div>
							</div>
						</div>
					{:else}<div class="info-row">{msg.contents}</div>{/if}
				{/each}
			</div>

			<div class="input-bar">
				<input
					bind:value={inputs.message}
					placeholder="Message the chat (or /leave to exit)"
					on:keydown={inputKeydown}
				/>
			</div>
		</div>
	{:else}
		<div class="connect-ui">
			<div class="mode-toggle">
				<button
					class:active={!inputs.makeNew}
					on:click={() => {
						inputs.makeNew = false;
					}}>Join Group</button
				>
				<button
					class:active={inputs.makeNew}
					on:click={() => {
						inputs.makeNew = true;
						inputs.groupKey = "";
					}}>Create Group</button
				>
			</div>

			<div class="connect-field">
				<p style="margin-bottom: 3px;">Server</p>
				<input bind:value={inputs.domain} placeholder="example.com" />
			</div>

			<div class="connect-field">
				<p style="margin-bottom: 3px;">Group key</p>
				<input
					bind:value={inputs.groupKey}
					disabled={inputs.makeNew}
					placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
				/>
			</div>

			<button
				class="connect-btn"
				on:click={() => {
					Connect(inputs.domain, inputs.makeNew, inputs.groupKey);
					inputs.groupKey = "";
				}}>Connect</button
			>
		</div>
	{/if}
</main>
