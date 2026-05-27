export const opOpen = 1;
export const opPcm = 2;
export const opClose = 3;
export const opWake = 4;

export function readFrame(chunks: Buffer[], offset: number): { frame: Buffer | null; offset: number } {

  const flat = Buffer.concat(chunks).subarray(offset);

  if (flat.length < 3) return { frame: null, offset };

  const op = flat[0];
  const idLen = flat.readUInt16BE(1);

  if (op === opPcm) {

    if (flat.length < 3 + idLen + 4) return { frame: null, offset };

    const pcmLen = flat.readUInt32BE(3 + idLen);
    const frameLen = 3 + idLen + 4 + pcmLen;

    if (flat.length < frameLen) return { frame: null, offset };

    return { frame: flat.subarray(0, frameLen), offset: offset + frameLen };

  }

  const frameLen = 3 + idLen;

  if (flat.length < frameLen) return { frame: null, offset };

  return { frame: flat.subarray(0, frameLen), offset: offset + frameLen };

}

export function writeWake(streamId: string): Buffer {

  const id = Buffer.from(streamId, "utf8");
  const out = Buffer.allocUnsafe(3 + id.length);

  out[0] = opWake;
  out.writeUInt16BE(id.length, 1);
  id.copy(out, 3);

  return out;

}
