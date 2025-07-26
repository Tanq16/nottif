document.addEventListener('DOMContentLoaded', () => {
    // --- DOM Elements ---
    const cronJobsList = document.getElementById('cron-jobs-list');
    const recentRequestsList = document.getElementById('recent-requests-list');
    const addCronForm = document.getElementById('add-cron-form');
    const updateWebhookForm = document.getElementById('update-webhook-form');
    const webhookUrlInput = document.getElementById('webhook_url');
    const testNotificationBtnDesktop = document.getElementById('test-notification-btn-desktop');
    const testNotificationBtnMobile = document.getElementById('test-notification-btn-mobile');

    // --- Toast Notification ---
    const showToast = (message, isError = false) => {
        const toast = document.createElement('div');
        toast.textContent = message;
        toast.className = 'fixed bottom-5 right-5 p-4 rounded-lg shadow-lg text-white';
        toast.style.backgroundColor = isError ? 'var(--red)' : 'var(--green)';
        toast.style.color = 'var(--crust)';
        document.body.appendChild(toast);
        setTimeout(() => {
            toast.style.transition = 'opacity 0.5s ease';
            toast.style.opacity = '0';
            setTimeout(() => document.body.removeChild(toast), 500);
        }, 3000);
    };

    // --- API Functions & Rendering ---

    const loadCronJobs = async () => {
        try {
            const response = await fetch('/api/cron/list');
            if (!response.ok) throw new Error('Failed to fetch cron jobs');
            const jobs = await response.json();
            renderCronJobs(jobs);
        } catch (error) {
            console.error('Error loading cron jobs:', error);
            cronJobsList.innerHTML = `<p class="p-2 text-center" style="color: var(--red);">Error loading jobs.</p>`;
        }
    };

    const renderCronJobs = (jobs) => {
        cronJobsList.innerHTML = '';
        if (jobs && jobs.length > 0) {
            jobs.forEach(job => {
                const jobElement = document.createElement('div');
                jobElement.className = 'flex items-center p-2 rounded-lg';
                jobElement.dataset.id = job.id;
                jobElement.innerHTML = `
                    <div class="flex items-center justify-center w-10 h-10 rounded-full mr-4 text-lg flex-shrink-0" style="background-color: var(--surface0); color: var(--sky);">
                        <i class="fa-solid fa-clock"></i>
                    </div>
                    <div class="flex-grow">
                        <p class="font-semibold">${job.message}</p>
                        <p class="text-sm font-mono" style="color: var(--subtext0);">${job.schedule}</p>
                    </div>
                    <button class="delete-cron-btn text-overlay1 hover:text-red transition-colors p-1 text-lg" title="Delete Cron Job">
                        <i class="fa-solid fa-trash fa-fw"></i>
                    </button>
                `;
                jobElement.querySelector('.delete-cron-btn').addEventListener('click', () => deleteCronJob(job.id));
                cronJobsList.appendChild(jobElement);
            });
        } else {
            cronJobsList.innerHTML = `<p class="p-2 text-center" style="color: var(--overlay0);">No scheduled jobs.</p>`;
        }
    };

    const loadEvents = async () => {
        try {
            const response = await fetch('/api/events');
            if (!response.ok) throw new Error('Failed to fetch events');
            const events = await response.json();
            renderEvents(events);
        } catch (error) {
            console.error('Error loading events:', error);
            recentRequestsList.innerHTML = `<p class="p-2 text-center" style="color: var(--red);">Error loading events.</p>`;
        }
    };

    const createEventElement = (event) => {
        const iconColor = event.success ? 'var(--green)' : 'var(--red)';
        const iconClass = event.success ? 'fa-solid fa-circle-check' : 'fa-solid fa-circle-exclamation';
        const time = new Date(event.timestamp).toLocaleTimeString();
        
        // Message formatting is now handled by the backend.
        const messageText = event.message;
        
        const eventElement = document.createElement('div');
        eventElement.className = 'flex items-center p-2 rounded-lg';
        eventElement.innerHTML = `
            <div class="flex items-center justify-center w-10 h-10 rounded-full mr-4 flex-shrink-0 text-xl" style="background-color: var(--surface0); color: ${iconColor};">
                <i class="${iconClass}"></i>
            </div>
            <p class="font-semibold flex-grow">${event.source}: ${messageText}</p>
            <p class="text-sm font-mono" style="color: var(--subtext0);">${time}</p>
        `;
        return eventElement;
    };

    const renderEvents = (events) => {
        recentRequestsList.innerHTML = '';
        if (events && events.length > 0) {
            events.forEach(event => {
                recentRequestsList.appendChild(createEventElement(event));
            });
        } else {
            recentRequestsList.innerHTML = `<p class="p-2 text-center" style="color: var(--overlay0);">No recent notifications.</p>`;
        }
    };

    const addCronJob = async (e) => {
        e.preventDefault();
        const message = document.getElementById('cron_message').value;
        const schedule = document.getElementById('cron_schedule').value;

        try {
            const response = await fetch('/api/cron/add', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message, schedule }),
            });
            if (!response.ok) throw new Error('Failed to add cron job');
            addCronForm.reset();
            showToast('Cron job added successfully!');
            loadCronJobs(); // Refresh the list immediately for better UX
        } catch (error) {
            console.error('Error adding cron job:', error);
            showToast('Error: Could not add cron job.', true);
        }
    };

    const deleteCronJob = async (id) => {
        try {
            const response = await fetch(`/api/cron/delete/${id}`, { method: 'DELETE' });
            if (!response.ok) throw new Error('Failed to delete cron job');
            showToast('Cron job deleted.');
            loadCronJobs(); // Refresh the list immediately for better UX
        } catch (error) {
            console.error('Error deleting cron job:', error);
            showToast('Error: Could not delete cron job.', true);
        }
    };
    
    const updateWebhook = async (e) => {
        e.preventDefault();
        const url = webhookUrlInput.value;
        try {
            const response = await fetch('/api/webhook/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url }),
            });
            if (!response.ok) throw new Error('Failed to update webhook URL');
            showToast('Webhook URL updated successfully!');
        } catch (error) {
            console.error('Error updating webhook:', error);
            showToast('Error: Could not update webhook URL.', true);
        }
    };

    const sendTestNotification = async () => {
        try {
            const response = await fetch('/api/webhook/test', { method: 'POST' });
            const result = await response.json();
            if (!response.ok) throw new Error(result.error || 'Failed to send');
            showToast('Test notification sent!');
        } catch (error) {
            console.error('Error sending test notification:', error);
            showToast(`Error: ${error.message}`, true);
        }
    };

    const connectSSE = () => {
        const evtSource = new EventSource('/api/events/stream');

        evtSource.onmessage = (e) => {
            // The data is a JSON array of all recent events. The arrival of any
            // message from the server signifies a state change.
            const allEvents = JSON.parse(e.data);

            // Re-render the events list with the data we just received.
            renderEvents(allEvents);

            // Always reload the cron jobs list to ensure it's in sync.
            loadCronJobs();
        };

        evtSource.onerror = () => {
            console.log("SSE connection closed or errored. Will not reconnect automatically.");
            evtSource.close();
        };
    };

    // --- Event Listeners ---
    addCronForm.addEventListener('submit', addCronJob);
    updateWebhookForm.addEventListener('submit', updateWebhook);
    testNotificationBtnDesktop.addEventListener('click', sendTestNotification);
    testNotificationBtnMobile.addEventListener('click', sendTestNotification);

    // --- Initial Load & SSE Connection ---
    loadCronJobs();
    loadEvents();
    connectSSE();
});
