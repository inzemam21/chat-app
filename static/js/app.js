let ws = null;
let typingTimeout = null;
let username = '';

function getElement(id) {
    const elem = document.getElementById(id);
    if (!elem) console.error(`Element #${id} not found - Check index.html or browser extensions`);
    return elem;
}

function joinRoom() {
    username = getElement('username-input')?.value.trim();
    const roomID = getElement('room-input')?.value.trim();
    if (!roomID || !username) return;

    const messages = getElement('messages');
    const input = getElement('message-input');
    const typingIndicator = getElement('typing-indicator');
    const typingUser = getElement('typing-user');
    const roomInfo = getElement('room-info');
    const otherStatus = getElement('other-status');
    const otherName = getElement('other-name');
    const roomForm = getElement('room-form');
    const chat = getElement('chat');
    const messageForm = getElement('message-form');

    if (!otherName || !otherStatus) {
        console.error('Critical DOM elements missing: other-name or other-status not found.');
        return;
    }

    if (ws && ws.readyState === WebSocket.OPEN) ws.close();

    ws = new WebSocket(`ws://localhost:8080/ws?room=${roomID}&username=${encodeURIComponent(username)}`);

    ws.onmessage = (event) => {
        const data = event.data;
        if (data.startsWith('typing:')) {
            const parts = data.split(':');
            const isTyping = parts[1] === '1';
            const typingUsername = parts[2] || '';
            typingIndicator.style.display = isTyping ? 'block' : 'none';
            typingUser.textContent = isTyping && typingUsername !== username ? `${typingUsername} is typing` : '';
        } else if (data.startsWith('status:')) {
            const parts = data.split(':');
            if (parts.length !== 3) return;
            const otherUser = parts[1];
            const status = parts[2];
            otherName.textContent = otherUser;
            otherStatus.className = `status-icon ${status.toLowerCase()}`;
        } else if (data.startsWith('system:')) {
            const formattedTime = new Date().toLocaleString('en-US', {
                day: 'numeric',
                month: 'long',
                hour: 'numeric',
                minute: '2-digit',
                hour12: true
            }).replace(',', '');
            const systemMessage = data.split('system:')[1];
            addMessage(formattedTime, `System: ${systemMessage}`, false);
        } else {
            const parts = data.split('|', 2);
            const timestamp = parts[0];
            const messageContent = parts[1] || data;
            const isSent = messageContent.startsWith(`${username}: `); // Check if sent by current user
            const formattedTime = new Date().toLocaleString('en-US', {
                day: 'numeric',
                month: 'long',
                hour: 'numeric',
                minute: '2-digit',
                hour12: true
            }).replace(',', '');
            addMessage(formattedTime, messageContent, isSent);
        }
    };

    ws.onopen = () => {
        roomForm.style.display = 'none';
        chat.style.display = 'flex';
        roomInfo.textContent = `#${roomID}`;
        otherName.textContent = 'Other';
        otherStatus.className = 'status-icon offline';
    };

    ws.onclose = () => {
        roomForm.style.display = 'flex';
        chat.style.display = 'none';
        typingIndicator.style.display = 'none';
        otherStatus.className = 'status-icon offline';
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };

    input.oninput = () => {
        if (ws) {
            ws.send('typing:1');
            clearTimeout(typingTimeout);
            typingTimeout = setTimeout(() => {
                if (ws) ws.send('typing:0');
            }, 1000);
        }
    };

    messageForm.onsubmit = (e) => {
        e.preventDefault();
        if (ws && input.value.trim() !== '') {
            const messageText = input.value;
            const timestamp = new Date().toLocaleString('en-US', {
                day: 'numeric',
                month: 'long',
                hour: 'numeric',
                minute: '2-digit',
                hour12: true
            }).replace(',', '');
            const messageContent = `${username}: ${messageText}`; // Still send with username for backend
            ws.send(messageText);
            addMessage(timestamp, messageText, true); // Pass only message text to frontend
            input.value = '';
            ws.send('typing:0');
            clearTimeout(typingTimeout);
        }
    };
}