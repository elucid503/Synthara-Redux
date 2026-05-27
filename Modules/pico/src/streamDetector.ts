import { Porcupine } from "@picovoice/porcupine-node";

export class StreamDetector {

  private readonly porcupine: Porcupine;
  private readonly frameLength: number;

  private pending = new Int16Array(0);
  private pendingLen = 0;

  constructor(modelPath: string, sensitivity: number) {

    this.porcupine = new Porcupine("None", [modelPath], [sensitivity]);
    this.frameLength = this.porcupine.frameLength;

  }

  feed(pcm: Int16Array): boolean {

    if (pcm.length === 0) {

      return false;

    }

    const merged = new Int16Array(this.pendingLen + pcm.length);
    merged.set(this.pending.subarray(0, this.pendingLen), 0);
    merged.set(pcm, this.pendingLen);

    let offset = 0;

    while (offset + this.frameLength <= merged.length) {

      const frame = merged.subarray(offset, offset + this.frameLength);
      const index = this.porcupine.process(frame);

      if (index >= 0) {

        this.storePending(merged.subarray(offset + this.frameLength));
        return true;

      }

      offset += this.frameLength;

    }

    this.storePending(merged.subarray(offset));
    return false;

  }

  close(): void {

    this.porcupine.release();

  }

  private storePending(samples: Int16Array): void {

    if (samples.length === 0) {

      this.pendingLen = 0;
      return;

    }

    if (this.pending.length < samples.length) {

      this.pending = new Int16Array(Math.max(samples.length, this.frameLength * 2));

    }

    this.pending.set(samples, 0);
    this.pendingLen = samples.length;

  }

}
