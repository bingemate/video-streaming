import express from 'express';
import path from 'path';
import {ChildProcessWithoutNullStreams, spawn} from 'child_process';
import ffmpeg from 'fluent-ffmpeg';
import fs from 'fs';
import cors from 'cors';

const app = express();
const port = 8001;

// Add cors
app.use(cors(
    {
        origin: '*'
    }
));

// Chemin vers le fichier vidéo local
const videoFilePath = '/home/nospy/Téléchargements/media/file.mkv';

async function createStream(req: express.Request, res: express.Response) {
    // Vérification si le fichier vidéo existe
    if (!fs.existsSync(videoFilePath)) {
        return res.status(404).send('File not found');
    }

    const range = req.headers.range;
    const fileSize = fs.statSync(videoFilePath).size;

    // Si la plage n'est pas spécifiée, envoyer tout le fichier
    if (!range) {
        return res.status(200).sendFile(videoFilePath);
    }

    // Analyser la plage de lecture
    const CHUNK_SIZE = 10 ** 6; // 1MB
    const start = Number(range.replace(/\D/g, ''));
    const end = Math.min(start + CHUNK_SIZE, fileSize - 1);

    // Modifier les en-têtes pour prendre en compte les parties de la vidéo à envoyer
    const contentLength = end - start + 1;
    const headers = {
        'Content-Range': `bytes ${start}-${end}/${fileSize}`,
        'Accept-Ranges': 'bytes',
        'Content-Length': contentLength,
        'Content-Type': 'video/mp4',
    };

    // Envoyer les en-têtes
    res.writeHead(206, headers);

    // Renvoyer le flux vidéo
    const stream = fs.createReadStream(videoFilePath, {start, end});

    stream.pipe(res);
}

// Démarrer le flux vidéo en utilisant GStreamer
let gstPipeline: ChildProcessWithoutNullStreams;

function startStream() {
    const gstArgs = [
        '-v',
        'tcpclientsrc host=localhost port=5000 ! matroskademux ! queue ! x264enc ! mpegtsmux ! queue ! tcpserversink host=localhost port=8001',
    ];

    gstPipeline = spawn('gst-launch-1.0', gstArgs);
}

function pauseStream() {
    if (gstPipeline) {
        gstPipeline.kill('SIGSTOP');
    }
}

function resumeStream() {
    if (gstPipeline) {
        gstPipeline.kill('SIGCONT');
    }
}

function seekStream(timeInSeconds: number) {
    const ffmpegCommand = ffmpeg(videoFilePath);
    ffmpegCommand
        .seekInput(timeInSeconds)
        .output('/dev/null')
        .on('start', () => {
            console.log(`Seeking to ${timeInSeconds}s`);
        })
        .on('end', () => {
            console.log('Seeking finished');
        })
        .run();
}

function stopStream() {
    if (gstPipeline) {
        gstPipeline.kill('SIGTERM');
    }
}

// Routes
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/video', createStream);

app.post('/stream/start', (req, res) => {
    startStream();
    res.sendStatus(200);
});

app.post('/stream/pause', (req, res) => {
    pauseStream();
    res.sendStatus(200);
});

app.post('/stream/resume', (req, res) => {
    resumeStream();
    res.sendStatus(200);
});

app.post('/stream/seek', (req, res) => {
    const {time} = req.body;
    seekStream(time);
    res.sendStatus(200);
});

app.post('/stream/stop', (req, res) => {
    stopStream();
    res.sendStatus(200);
});

// Démarrer le serveur
app.listen(port, () => {
    console.log(`Server running on port ${port}`);
});
