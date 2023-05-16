const videoPlayer = document.querySelector('#videoPlayer');
const playButton = document.querySelector('#play');
const pauseButton = document.querySelector('#pause');
const seekInput = document.querySelector('#seek');
const stopButton = document.querySelector('#stop');

// Gérer les événements du vidéo player
videoPlayer.addEventListener('play', () => {
    sendStreamAction('start');
});

videoPlayer.addEventListener('pause', () => {
    sendStreamAction('pause');
});

videoPlayer.addEventListener('seeked', () => {
    sendStreamAction('seek', { time: videoPlayer.currentTime });
});

// Gérer les événements des boutons de contrôle
playButton.addEventListener('click', () => {
    videoPlayer.play();
});

pauseButton.addEventListener('click', () => {
    videoPlayer.pause();
});

seekInput.addEventListener('input', () => {
    videoPlayer.currentTime = seekInput.value;
});

stopButton.addEventListener('click', () => {
    videoPlayer.pause();
    videoPlayer.currentTime = 0;
    sendStreamAction('stop');
});

function sendStreamAction(action, data = null) {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', `http://localhost:8001/stream/${action}`, true);
    if (data) {
        // set withCredentials to true to send cookies in CORS request
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.send(JSON.stringify(data));
    } else {
        xhr.send();
    }
}
