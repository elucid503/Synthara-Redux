# Voice Command Models

On-device keyword spotting uses [sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx) to detect the wake word **Synthara** before audio is sent to xAI for transcription.

## Required Files

Default directory: `./Receive/Models/kws/` (relative to the process working directory).

| Env Var | Default |
|---------|---------|
| `KWS_ENCODER` | `./Receive/Models/kws/encoder.onnx` |
| `KWS_DECODER` | `./Receive/Models/kws/decoder.onnx` |
| `KWS_JOINER` | `./Receive/Models/kws/joiner.onnx` |
| `KWS_TOKENS` | `./Receive/Models/kws/tokens.txt` |
| `KWS_KEYWORDS_FILE` | `./Receive/Models/kws/keywords.txt` |
| `KWS_BPE_VOCAB` | `./Receive/Models/kws/bpe.model` |

Optional tuning: `KWS_KEYWORDS_SCORE`, `KWS_KEYWORDS_THRESHOLD`, `KWS_MAX_ACTIVE_PATHS`, `KWS_BATCH_MS`, `VOICE_COMMANDS=false` to disable.

If any file is missing, wake-word detection is skipped and voice commands stay off; the rest of the bot still runs.

## Download

```bash
cd Receive/Models
wget https://github.com/k2-fsa/sherpa-onnx/releases/download/kws-models/sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01.tar.bz2
tar xvf sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01.tar.bz2
mv sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01/encoder-epoch-12-avg-2-chunk-16-left-64.onnx kws/encoder.onnx
mv sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01/decoder-epoch-12-avg-2-chunk-16-left-64.onnx kws/decoder.onnx
mv sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01/joiner-epoch-12-avg-2-chunk-16-left-64.onnx kws/joiner.onnx
mv sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01/tokens.txt kws/tokens.txt
mv sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01/bpe.model kws/bpe.model
rm -rf sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01.tar.bz2 sherpa-onnx-kws-zipformer-gigaspeech-3.3M-2024-01-01
```

Place your custom wake phrases in `kws/keywords.txt` (see [sherpa-onnx KWS docs](https://k2-fsa.github.io/sherpa/onnx/kws/index.html)). The value after `#` must be a float, not a name.
