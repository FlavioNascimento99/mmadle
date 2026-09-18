import { Container, getContainer } from "@cloudflare/containers";

interface Env {
  MMADLE_API: DurableObjectNamespace<MmadleApi>;
  DATABASE_URL: string;
  GAME_TIMEZONE: string;
}

export class MmadleApi extends Container<Env> {
  defaultPort = 8080;
  sleepAfter = "10m";
  enableInternet = true;

  constructor(ctx: DurableObjectState<Env>, env: Env) {
    super(ctx, env);
    this.envVars = {
      DATABASE_URL: env.DATABASE_URL,
      GAME_TIMEZONE: env.GAME_TIMEZONE,
    };
  }
}

// Static assets are served before the Worker runs; only /api/* reaches here
// (see assets.run_worker_first), so every request belongs to the backend.
export default {
  fetch(request, env) {
    return getContainer(env.MMADLE_API).fetch(request);
  },
} satisfies ExportedHandler<Env>;
