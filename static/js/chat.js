function addMessage(timestamp, content, isSent) {
    const messages = document.getElementById('messages');
    if (!messages) {
        console.error('Messages container not found');
        return;
    }

    const messageText = content.split(': ', 2)[1] || content; // Extract message text after ": ", fallback to full content

    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${isSent ? 'sent' : 'received'}`;

    const messageWrapper = document.createElement('div');
    messageWrapper.className = 'message-wrapper';

    const contentSpan = document.createElement('span');
    contentSpan.className = 'content';
    contentSpan.textContent = messageText;

    const timeSpan = document.createElement('span');
    timeSpan.className = 'timestamp';
    timeSpan.textContent = timestamp;

    messageWrapper.appendChild(contentSpan);
    messageDiv.appendChild(messageWrapper);
    messageDiv.appendChild(timeSpan);
    messages.appendChild(messageDiv);

    // Auto-scroll to latest message
    messages.scrollTop = messages.scrollHeight;
}