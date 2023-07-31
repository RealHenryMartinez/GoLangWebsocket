// selectedchat is by default General.
var selectedchat = "general";

/**
 * Event is used to wrap all messages Send and Recieved
 * on the Websocket
 * The type is used as a RPC
 * */
class Event {
  // Each Event needs a Type
  // The payload is not required
  constructor(type, payload) {
    this.type = type;
    this.payload = payload;
  }
}
/**
 * SendMessageEvent is used to send messages to other clients
 * */
class SendMessageEvent {
  constructor(message, from) {
    this.message = message;
    this.from = from;
  }
}
/**
 * NewMessageEvent is messages comming from clients
 * */
class NewMessageEvent {
  constructor(message, from, sent) {
    this.message = message;
    this.from = from;
    this.sent = sent;
  }
}

/**
 * routeEvent is a proxy function that routes
 * events into their correct Handler
 * based on the type field
 * */
function routeEvent(event) {
  if (event.type === undefined) {
    alert("no 'type' field in event");
  }
  switch (event.type) {
    case "new_message":
      // Format payload
      const messageEvent = Object.assign(new NewMessageEvent(), event.payload);
      appendChatMessage(messageEvent);
      break;
    default:
      alert("unsupported message type");
      break;
  }
}
/**
 * appendChatMessage takes in new messages and adds them to the chat
 * */
function appendChatMessage(messageEvent) {
  var date = new Date(messageEvent.sent);
  // format message
  const formattedMsg = `${date.toLocaleString()}: ${messageEvent.message}`;
  // Append Message
  textarea = document.getElementById("chatmessages");
  textarea.innerHTML = textarea.innerHTML + "\n" + formattedMsg;
  textarea.scrollTop = textarea.scrollHeight;
}

/**
 * ChangeChatRoomEvent is used to switch chatroom
 * */
class ChangeChatRoomEvent {
  constructor(name) {
    this.name = name;
  }
}
/**
 * changeChatRoom will update the value of selectedchat
 * and also notify the server that it changes chatroom
 * */
function changeChatRoom() {
  // Change Header to reflect the Changed chatroom
  var newchat = document.getElementById("chatroom");
  if (newchat != null && newchat.value != selectedchat) {
    selectedchat = newchat.value;
    header = document.getElementById("chat-header").innerHTML =
      "Currently in chat: " + selectedchat;

    let changeEvent = new ChangeChatRoomEvent(selectedchat);
    sendEvent("change_room", changeEvent);
    textarea = document.getElementById("chatmessages");
    textarea.innerHTML = `You changed room into: ${selectedchat}`;
  }
  return false;
}
/**
 * sendMessage will send a new message onto the Chat
 * */
// Sending a message with the correct payload format
function sendMessage(e) {
  e.preventDefault();
  var newmessage = document.getElementById("message");
  if (newmessage != null) {
    console.log(newmessage);

    // Construct the payload as a JSON object
    const payload = {
      type: "send_message",
      payload: {
        message: newmessage.value,
        from: "your_username", // Replace "your_username" with the actual username of the sender
      },
    };

    // Send the event to the WebSocket connection
    conn.send(JSON.stringify(payload));
  }
  return false;
}


/**
 * sendEvent
 * eventname - the event name to send on
 * payload - the data payload
 * */
function sendEvent(eventName, payload) {
  // Create a event Object with a event named send_message
  const event = new Event(eventName, payload);
  // Format as JSON and send
  conn.send(JSON.stringify(event));
}
/**
 * login will send a login request to the server and then
 * connect websocket
 * */
function login() {
  let formData = {
    username: document.getElementById("username").value,
    password: document.getElementById("password").value,
  };
  // Send the request
  fetch("login", {
    method: "post",
    body: JSON.stringify(formData),
    mode: "cors",
  })
    .then((response) => {
      if (response.ok) {
        return response.json();
      } else {
        throw "unauthorized";
      }
    })
    .then((data) => {
      // Now we have a OTP, send a Request to Connect to WebSocket
      connectWebsocket(data.otp);
    })
    .catch((e) => {
      alert(e);
    });
  return false;
}
/**
 * ConnectWebsocket will connect to websocket and add listeners
 * */
function connectWebsocket(otp) {
  // Check if the browser supports WebSocket
  if (window["WebSocket"]) {
    console.log("supports websockets");
    // Connect to websocket using OTP as a GET parameter
    conn = new WebSocket("ws://" + document.location.host + "/ws?otp=" + otp);

    // Onopen
    conn.onopen = function (evt) {
      document.getElementById("connection-header").innerHTML =
        "Connected to Websocket: true";
    };

    conn.onclose = function (evt) {
      // Set disconnected
      document.getElementById("connection-header").innerHTML =
        "Connected to Websocket: false";
    };

    // Add a listener to the onmessage event
    conn.onmessage = function (evt) {
      console.log(evt);
      // parse websocket message as JSON
      const eventData = JSON.parse(evt.data);
      // Assign JSON data to new Event Object
      const event = Object.assign(new Event(), eventData);
      // Let router manage message
      routeEvent(event);
    };
  } else {
    alert("Not supporting websockets");
  }
}
/**
 * Once the website loads
 * */
window.onload = function () {
  // Apply our listener functions to the submit event on both forms
  // we do it this way to avoid redirects
  document.getElementById("chatroom-selection").onsubmit = changeChatRoom;
  document.getElementById("chatroom-message").onsubmit = sendMessage;
  document.getElementById("login-form").onsubmit = login;
};
