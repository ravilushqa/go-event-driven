document.addEventListener('DOMContentLoaded', function() {
    // Setup event listeners for HTMX events
    document.body.addEventListener('htmx:afterSwap', function(event) {
        // Handle successful operations that may need UI updates
        if (event.detail.target.id === 'notification-area') {
            // Auto-refresh messages list after a notification is shown
            setTimeout(function() {
                document.querySelector('#message-list').setAttribute('hx-get', '/messages');
                document.querySelector('#message-list').setAttribute('hx-trigger', 'load');
            }, 500);
        }
    });

    // Listen for the custom refreshMessages event
    document.body.addEventListener('refreshMessages', function() {
        document.querySelector('#message-list').setAttribute('hx-get', '/messages');
        document.querySelector('#message-list').setAttribute('hx-trigger', 'load');
    });
});