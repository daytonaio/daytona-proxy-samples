import express from 'express';
import type { Request, Response } from 'express';
import * as dotenv from 'dotenv';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { Configuration, SandboxApi } from '@daytonaio/api-client';
import path from 'path';
import { ClientRequest } from 'http';

dotenv.config({ quiet: true });

const DAYTONA_API_KEY = process.env.DAYTONA_API_KEY;
const DAYTONA_API_URL = process.env.DAYTONA_API_URL;

if (!DAYTONA_API_KEY) {
  throw new Error('DAYTONA_API_KEY is not set');
}

if (!DAYTONA_API_URL) {
  throw new Error('DAYTONA_API_URL is not set');
}

const sandboxApi = new SandboxApi(new Configuration({
  basePath: DAYTONA_API_URL,
  baseOptions: {
    headers: {
      'Authorization': `Bearer ${DAYTONA_API_KEY}`
    }
  }
}))

function getSandboxIdAndPortFromUrl(url: string) {
  if (url.split('.').length === 1) {
    throw new Error('Invalid URL');
  }

  const subdomain = url.split('.')[0]
  const hostname = url.split('.').slice(1).join('.')

  if (subdomain.split('-').length < 2) {
    throw new Error('Invalid URL');
  }

  const port = subdomain.split('-')[0]
  const sandboxId = subdomain.split('-').slice(1).join('-')

  return { sandboxId, port: parseInt(port), hostname };
}

const onReq = (proxyReq: ClientRequest, req: Request, onError: (error: any) => void) => {
  // @ts-expect-error
  const err = req._err
  if (err) {
    console.log('error', err.data?.message || String(err))
    onError(err)
    return
  }

  // @ts-expect-error
  if (req._authToken) {
    try {
      // @ts-expect-error
      proxyReq.setHeader('X-Daytona-Preview-Token', req._authToken)
    } catch (error) {
      console.log('error', error)
    }
  }
}

const proxyMiddleware = createProxyMiddleware<Request, Response>({
  router: async (req) => {
    try {
      if (!req.headers.host) {
        throw new Error('Invalid URL. Host is required');
      }

      const { sandboxId, port } = getSandboxIdAndPortFromUrl(req.headers.host);

      // Get sandbox preview URL
      // If organization doesn't own the sandbox, this will throw an error
      // NOTE: This API call should be cached.
      // Otherwise, all traffic going to the proxy will trigger this API call and be slowed down
      const response = await sandboxApi.getPortPreviewUrl(sandboxId, port)

      // @ts-expect-error
      req._authToken = response.data.token

      return response.data.url
    } catch (error) {
      // @ts-expect-error
      req._err = error
    }

    // Must return a valid URL
    return 'http://target-error'
  },
  changeOrigin: true,
  autoRewrite: true,
  ws: true,
  headers: {
    // Set if you want to disable Daytona CORS
    // "X-Daytona-Disable-CORS": "true"
  },
  xfwd: true,
  on: {
    proxyReq: (proxyReq, req, res) => {
      onReq(proxyReq, req, (err) => {
        res.status(500).sendFile(path.join(__dirname, 'error.html'))
      })
    },
    proxyReqWs: (proxyReq, req, socket) => {
      onReq(proxyReq, req, (err) => {
        socket.end()
      })
    },
    proxyRes: (proxyRes, req, res) => {
      // Custom error handling
      if (proxyRes.statusCode !== 200) {
        res.status(500).sendFile(path.join(__dirname, 'error.html'))
      }
    }
  }
});

const app = express();

app.use(proxyMiddleware);

app.listen(process.env.PORT || 1234);

console.log(`Proxy server is running on port ${process.env.PORT || 1234}`);