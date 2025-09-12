import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';
import { credentials } from '@grpc/grpc-js';
import fs from 'fs';
import os from 'os';

const endpoint = process.env.PS_TELEMETRY_ENDPOINT || 'otel-collector:4317';
const insecureEnv = (process.env.PS_OTEL_INSECURE || 'true').toLowerCase();
const insecure = insecureEnv === '' || insecureEnv === 'true' || insecureEnv === '1' || insecureEnv === 'yes';
const caFile = process.env.PS_OTEL_CA_FILE || '';
const clientCertFile = process.env.PS_OTEL_CLIENT_CERT_FILE || '';
const clientKeyFile = process.env.PS_OTEL_CLIENT_KEY_FILE || '';
const version = process.env.npm_package_version || 'dev';
const instance = process.env.PS_INSTANCE_ID || os.hostname();

const creds = (() => {
  if (insecure) return credentials.createInsecure();
  const root = caFile ? fs.readFileSync(caFile) : undefined;
  if (clientCertFile && clientKeyFile) {
    const privateKey = fs.readFileSync(clientKeyFile);
    const certChain = fs.readFileSync(clientCertFile);
    return credentials.createSsl(root, privateKey, certChain);
  }
  return credentials.createSsl(root);
})();

const traceExporter = new OTLPTraceExporter({ url: endpoint, credentials: creds });

const sdk = new NodeSDK({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: 'promptshield-bff',
    [SemanticResourceAttributes.SERVICE_VERSION]: version,
    [SemanticResourceAttributes.SERVICE_INSTANCE_ID]: instance,
  }),
  traceExporter,
  instrumentations: [
    getNodeAutoInstrumentations({
      '@opentelemetry/instrumentation-http': {
        ignoreIncomingRequestHook: (req) => {
          const path = req.url || '';
          return /^\/healthz$/.test(path) || /^\/readyz$/.test(path) || /^\/metrics$/.test(path);
        },
      },
    }),
  ],
});

try {
  sdk.start();
} catch (err) {
  // eslint-disable-next-line no-console
  console.error('OTel NodeSDK init failed:', err);
}

process.on('SIGTERM', async () => {
  try { await sdk.shutdown(); } catch (e) { /* ignore */ }
});

export default sdk;

