import { accessSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { opClose, opOpen, opPcm, readFrame, writeWake } from "./protocol.js";
import { StreamDetector } from "./streamDetector.js";

const rootDir = dirname(fileURLToPath(import.meta.url));
const modelPath = join(rootDir, "..", "model", "synthara.ppn");
const sensitivity = 0.7;

try {

  accessSync(modelPath);

} catch {

  process.stderr.write("pico: missing model at " + modelPath + "\n");
  process.exit(1);

}

const streams = new Map<string, StreamDetector>();

const stdinChunks: Buffer[] = [];
let stdinOffset = 0;

function streamIdFrom(frame: Buffer): string {

  const idLen = frame.readUInt16BE(1);
  return frame.subarray(3, 3 + idLen).toString("utf8");

}

function pcmFrom(frame: Buffer): Int16Array {

  const idLen = frame.readUInt16BE(1);
  const pcmLen = frame.readUInt32BE(3 + idLen);

  const start = 7 + idLen;

  const bytes = frame.subarray(start, start + pcmLen);
  const samples = new Int16Array(bytes.length / 2);

  for (let i = 0; i < samples.length; i++) {

    samples[i] = bytes.readInt16LE(i * 2);

  }

  return samples;
}

function openStream(id: string): void {

  if (streams.has(id)) {

    return;

  }

  streams.set(id, new StreamDetector(modelPath, sensitivity));

}

function closeStream(id: string): void {

  const detector = streams.get(id);

  if (detector) {

    detector.close();
    streams.delete(id);

  }

}

function feedStream(id: string, pcm: Int16Array): void {

  let detector = streams.get(id);

  if (!detector) {

    detector = new StreamDetector(modelPath, sensitivity);
    streams.set(id, detector);

  }

  if (detector.feed(pcm)) {

    process.stdout.write(writeWake(id));

  }

}

function handleFrame(frame: Buffer): void {

  const op = frame[0];
  const id = streamIdFrom(frame);

  switch (op) {

    case opOpen:

      openStream(id);
      break;

    case opPcm:

      feedStream(id, pcmFrom(frame));
      break;

    case opClose:

      closeStream(id);
      break;

    default:

      break;

  }

}

function drainStdin(): void {

  while (true) {

    const { frame, offset } = readFrame(stdinChunks, stdinOffset);

    if (!frame) {

      break;

    }

    stdinOffset = offset;
    handleFrame(frame);

  }

  while (stdinOffset > 0 && stdinChunks.length > 0) {

    const first = stdinChunks[0];

    if (stdinOffset >= first.length) {

      stdinOffset -= first.length;
      stdinChunks.shift();

      continue;

    }

    stdinChunks[0] = first.subarray(stdinOffset);
    stdinOffset = 0;

    break;

  }

}

process.stdin.on("data", (chunk: Buffer) => {

  stdinChunks.push(chunk);
  drainStdin();

});

process.stdin.on("end", () => {

  for (const id of [...streams.keys()]) {

    closeStream(id);

  }

  process.exit(0);

});

process.stdin.resume();
