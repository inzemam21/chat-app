# Real-Time Chat App

A modern, real-time chat application inspired by Slack, built with Go (backend) and JavaScript (frontend). Users can join rooms, chat instantly, see typing indicators, and view status updates with a clean, eye-appealing design.

## Features

- **Real-Time Messaging**: Instant chat via WebSockets.
- **Typing Indicators**: Animated "X is typing..." display.
- **User Status**: Online/offline with green/gray dots.
- **Modern UI**: No usernames in messages, "7 March 8:54 PM" timestamps, sleek bubbles (blue for sent, gray for received).
- **System Notices**: Single "asda joined the room" messages, centered and italicized.
- **Two-User Rooms**: Limited to two users per room (expandable).
- **Responsive**: Works on Chrome and Edge (disable interfering extensions).

## Tech Stack

- **Backend**: Go with `gorilla/websocket` for WebSocket support.
- **Frontend**: HTML/CSS (Slack-inspired, dark purple join form, light chat area), JavaScript (`app.js` for WebSocket logic, `chat.js` for rendering).
- **Deployment**: Local server on `localhost:8080`.

## Project Structure

chatapp/
│
├── main.go              # Sets up HTTP server and WebSocket routes
├── hub.go               # Manages WebSocket connections, rooms, and messages
├── static/              # Frontend assets
│   ├── index.html       # Main HTML file
│   ├── js/
│   │   ├── app.js       # WebSocket logic and core functionality
│   │   └── chat.js      # Message rendering
│   └── css/
│       ├── base.css     # Base styles and layout
│       ├── room-form.css # Styles for the join form
│       ├── chat.css     # Styles for the chat interface
│       └── animations.css # Typing indicator animations
├── README.md            # Project documentation


## Installation

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/<your-username>/chatapp.git
   cd chatapp

2. **Install Dependencies: Ensure Go is installed, then**:
  ``` go get github.com/gorilla/websocket

2. **Run application**:
  ```go run .
