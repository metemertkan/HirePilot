<script lang="ts">
    import { goto } from '$app/navigation';
    import { onMount, onDestroy } from 'svelte';

    let pollingInterval: NodeJS.Timeout | null = null;
    let hasPendingItems = false;
    let websocket: WebSocket | null = null;
    let connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error' = 'disconnected';
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;

    onMount(async () => {
        await fetchPrompts();
        connectWebSocket();
        startPollingIfNeeded();
    });

    // WebSocket connection for real-time updates
    function connectWebSocket() {
        if (reconnectAttempts >= maxReconnectAttempts) {
            console.log('Max reconnection attempts reached, falling back to polling');
            connectionStatus = 'error';
            startPollingIfNeeded();
            return;
        }

        try {
            connectionStatus = 'connecting';
            websocket = new WebSocket('ws://localhost:8080/ws');
            
            websocket.onopen = () => {
                console.log('WebSocket connected');
                connectionStatus = 'connected';
                reconnectAttempts = 0; // Reset on successful connection
            };
            
            websocket.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    handleWebSocketMessage(message);
                } catch (e) {
                    console.error('Error parsing WebSocket message:', e);
                }
            };
            
            websocket.onclose = () => {
                console.log('WebSocket disconnected');
                connectionStatus = 'disconnected';
                
                // Reconnect with exponential backoff
                const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
                reconnectAttempts++;
                
                setTimeout(connectWebSocket, delay);
            };
            
            websocket.onerror = (error) => {
                console.error('WebSocket error:', error);
                connectionStatus = 'error';
            };
        } catch (e) {
            console.error('Failed to connect WebSocket:', e);
            connectionStatus = 'error';
            // Fallback to polling if WebSocket fails
            startPollingIfNeeded();
        }
    }

    // Handle incoming WebSocket messages
    function handleWebSocketMessage(message: { type: string; data: any }) {
        switch (message.type) {
            case 'prompt_created':
                handlePromptCreated(message.data);
                break;
            case 'prompt_updated':
                handlePromptUpdated(message.data);
                break;
            default:
                console.log('Unknown WebSocket message type:', message.type);
        }
    }

    // Handle real-time prompt creation
    function handlePromptCreated(newPrompt: any) {
        // Remove any pending item with the same name
        prompts = prompts.filter(p => !(p.isPending && p.name === newPrompt.name));
        
        // Add the new prompt if it doesn't already exist
        if (!prompts.some(p => p.id === newPrompt.id)) {
            prompts = [...prompts, newPrompt];
        }
        
        updatePendingStatus();
    }

    // Handle real-time prompt updates
    function handlePromptUpdated(updatedPrompt: any) {
        prompts = prompts.map(p => 
            p.id === updatedPrompt.id ? updatedPrompt : p
        );
        updatePendingStatus();
    }

    // Update pending status and polling
    function updatePendingStatus() {
        hasPendingItems = prompts.some(p => p.isPending);
        if (!hasPendingItems && pollingInterval) {
            clearInterval(pollingInterval);
            pollingInterval = null;
        }
    }

    // Start polling when there are pending items (fallback)
    function startPollingIfNeeded() {
        if (hasPendingItems && !pollingInterval && (connectionStatus === 'error' || !websocket)) {
            pollingInterval = setInterval(async () => {
                await fetchPrompts();
                if (!hasPendingItems && pollingInterval) {
                    clearInterval(pollingInterval);
                    pollingInterval = null;
                }
            }, 2000);
        }
    }

    // Stop polling and close WebSocket when component is destroyed
    onDestroy(() => {
        if (pollingInterval) {
            clearInterval(pollingInterval);
        }
        if (websocket) {
            websocket.close();
        }
    });
         // Prompt state
    let prompts: { id: number; name: string; prompt: string, cvGenerationDefault: boolean, scoreGenerationDefault:boolean, coverGenerationDefault: boolean, isPending?: boolean }[] = [];
    let loadingPrompts = false;
    let errorPrompts = '';
    const PROMPT_API_URL = 'http://localhost:8080/api/prompts';
    let promptName = '';
    let promptText = '';
    let errorPrompt = '';
    let cvGenerationDefault = false;
    let scoreGenerationDefault = false;
    let coverGenerationDefault = false;

    async function addPrompt() {
        errorPrompt = '';
        
        // Create optimistic prompt object
        const optimisticPrompt = {
            id: Date.now(), // Temporary ID
            name: promptName,
            prompt: promptText,
            cvGenerationDefault: cvGenerationDefault,
            scoreGenerationDefault: scoreGenerationDefault,
            coverGenerationDefault: coverGenerationDefault,
            isPending: true // Flag to show it's pending
        };
        
        // Optimistically add to UI
        prompts = [...prompts, optimisticPrompt];
        
        // Clear form immediately
        const originalValues = { promptName, promptText, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault };
        promptName = '';
        promptText = '';
        cvGenerationDefault = false;
        scoreGenerationDefault = false;
        coverGenerationDefault = false;
        
        try {
            const res = await fetch(PROMPT_API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    name: originalValues.promptName, 
                    prompt: originalValues.promptText, 
                    cvGenerationDefault: originalValues.cvGenerationDefault, 
                    scoreGenerationDefault: originalValues.scoreGenerationDefault, 
                    coverGenerationDefault: originalValues.coverGenerationDefault 
                })
            });
            
            if (!res.ok) throw new Error('Failed to add prompt');
            
            // Wait a bit for the backend to process, then refresh
            setTimeout(async () => {
                await fetchPrompts();
            }, 1000);
            
        } catch (e) {
            // Remove optimistic prompt on error
            prompts = prompts.filter(p => p.id !== optimisticPrompt.id);
            
            // Restore form values
            promptName = originalValues.promptName;
            promptText = originalValues.promptText;
            cvGenerationDefault = originalValues.cvGenerationDefault;
            scoreGenerationDefault = originalValues.scoreGenerationDefault;
            coverGenerationDefault = originalValues.coverGenerationDefault;
            
            if (e instanceof Error) {
                errorPrompt = e.message;
            } else {
                errorPrompt = String(e);
            }
        }
    }
    async function fetchPrompts() {
        loadingPrompts = true;
        errorPrompts = '';
        try {
            const res = await fetch(PROMPT_API_URL);
            if (!res.ok) throw new Error('Failed to fetch prompts');
            const data = await res.json();
            
            // Preserve pending items that aren't in the response yet
            const pendingItems = prompts.filter(p => p.isPending);
            const newPrompts = Array.isArray(data) ? data : [];
            
            // Remove pending items that now exist in the response
            const filteredPendingItems = pendingItems.filter(pending => 
                !newPrompts.some(newPrompt => newPrompt.name === pending.name)
            );
            
            prompts = [...newPrompts, ...filteredPendingItems];
            hasPendingItems = filteredPendingItems.length > 0;
            
            // Start polling if we have pending items
            if (hasPendingItems) {
                startPollingIfNeeded();
            }
            
        } catch (e) {
            if (e instanceof Error) {
                errorPrompts = e.message;
            } else {
                errorPrompts = String(e);
            }            
        } finally {
            loadingPrompts = false;
        }
    }


</script>

<div class="d-flex justify-content-between align-items-center mb-3">
    <h2>Prompts</h2>
    <div class="connection-status">
        {#if connectionStatus === 'connected'}
            <span class="badge badge-success">🟢 Real-time</span>
        {:else if connectionStatus === 'connecting'}
            <span class="badge badge-warning">🟡 Connecting...</span>
        {:else if connectionStatus === 'disconnected'}
            <span class="badge badge-warning">🟡 Reconnecting...</span>
        {:else if connectionStatus === 'error'}
            <span class="badge badge-secondary">⚫ Polling mode</span>
        {/if}
    </div>
</div>


<div class="row justify-content-center">

    <div class="col-xl-10 col-lg-12 col-md-9">

        <div class="card o-hidden border-0 shadow-lg my-5">
            <div class="card-body p-0">
                <!-- Nested Row within Card Body -->
                <div class="row">
                    <div class="col-lg-12">
                        <div class="p-5">
                            <div class="text-center">
                                <h1 class="h4 text-gray-900 mb-4">Add A Prompt</h1>
                            </div>
                            <form class="user" on:submit|preventDefault={addPrompt}>
                                <div class="form-group">
                                    <input type="text" class="form-control form-control-user"
                                    bind:value={promptName} required placeholder="Prompt Name">
                                </div>
                                <div class="form-group">
                                    <textarea class="form-control form-control-user" placeholder="Prompt Text" 
                                    bind:value={promptText} required></textarea>
                                </div>
                                <div class="form-group">
                                    <label>
                                        CV Generation Default:
                                        <input type="checkbox" bind:checked={cvGenerationDefault} />
                                    </label>
                                </div>
                                <div class="form-group">
                                    <label>
                                        Score Generation Default:
                                        <input type="checkbox" bind:checked={scoreGenerationDefault} />
                                    </label>
                                </div>
                                <div class="form-group">
                                    <label>
                                        Cover Generation Default:
                                        <input type="checkbox" bind:checked={coverGenerationDefault} />
                                    </label>
                                </div>
                                <button class="btn btn-primary btn-user btn-block" type="submit">Add Prompt</button>
                                {#if errorPrompt}
                                    <div class="alert alert-danger mt-3">{errorPrompt}</div>
                                {/if}
                                {#if loadingPrompts}
                                    <div class="alert alert-info mt-3">Loading...</div>
                                {/if}
                                {#if errorPrompts}
                                    <div class="alert alert-danger mt-3">{errorPrompts}</div>
                                {/if}
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>




<div class="card shadow mb-4">
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-bordered" id="dataTable" width="100%" cellspacing="0">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Prompt</th>
                                <th>cvGenerationDefault</th>
                                <th>scoreGenerationDefault</th>
                                <th>coverGenerationDefault</th>
                                <th></th>
                            </tr>
                        </thead>
                        <tfoot>
                            <tr>
                                <th>Name</th>
                                <th>Prompt</th>
                                <th>cvGenerationDefault</th>
                                <th>scoreGenerationDefault</th>
                                <th>coverGenerationDefault</th>
                                <th></th>
                            </tr>
                        </tfoot>
                        <tbody>
                            {#each prompts as prompt}
                                <tr class={prompt.isPending ? 'table-warning' : ''}>
                                    <td>
                                        {prompt.name}
                                        {#if prompt.isPending}
                                            <span class="badge badge-warning ml-2">Processing...</span>
                                        {/if}
                                    </td>
                                    <td>{prompt.prompt.slice(0, 100)}{prompt.prompt.length > 100 ? '...' : ''}</td>
                                    <td>{prompt.cvGenerationDefault === '1' || prompt.cvGenerationDefault === true ? 'Yes' : 'No'}</td>
                                    <td>{prompt.scoreGenerationDefault === '1' || prompt.scoreGenerationDefault === true ? 'Yes' : 'No'}</td>
                                    <td>{prompt.coverGenerationDefault === '1' || prompt.coverGenerationDefault === true ? 'Yes' : 'No'}</td>
                                    <td>
                                        {#if !prompt.isPending}
                                            <button on:click={() => goto(`/prompts/${prompt.id}`)}>View</button>
                                        {:else}
                                            <button disabled class="btn btn-secondary btn-sm">Processing...</button>
                                        {/if}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>  