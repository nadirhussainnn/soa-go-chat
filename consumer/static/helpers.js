let SOCKET_EVENTS ={
    CONTACT_REQUEST_SENT_ACK : "CONTACT_REQUEST_SENT_ACK",
    NEW_CONTACT_REQUEST_RECEIVED : "NEW_CONTACT_REQUEST_RECEIVED",
    UPDATE_RECEIVED_ON_CONTACT_REQUEST : "UPDATE_RECEIVED_ON_CONTACT_REQUEST",
    UPDATE_SENT_ON_CONTACT_REQUEST : "UPDATE_SENT_ON_CONTACT_REQUEST"
}

function showSnackbar(message) {
    var x = document.getElementById("snackbar");
    x.className = "show";
    x.innerText = message
    setTimeout(function(){ x.className = x.className.replace("show", ""); }, 8000);
}