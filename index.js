import Fastify from 'fastify'
import { buscarEventos, reservarIngresso } from './db.js';

const app = Fastify({
  logger: false
})

app.get('/eventos', async (req, rep) => {
  const eventos = await buscarEventos();
  return eventos;
})

app.post('/reservas', async (req, rep) => {
  const body = req.body;
  const evento_id = body?.evento_id;
  const usuario_id = body?.usuario_id;

  if ((evento_id | 0) !== evento_id || (usuario_id | 0) !== usuario_id) {
    return rep.code(400).send();
  }

  const success = await reservarIngresso(evento_id, usuario_id);

  if (!success) {
    return rep.code(422).send();
  }

  return rep.code(201).send();
});

app.setErrorHandler((error, request, reply) => {
  const routeUrl = request.routeOptions?.url ?? request.routerPath;

  if (routeUrl === '/reservas' && error?.statusCode === 400) {
    return reply.code(400).send('Você mandou algo errado.')
  }

  return reply.code(error?.statusCode ?? 500).send();
})

try {
  const port = Number.parseInt(process.env.APP_PORT ?? '8080', 10) || 8080;
  await app.listen({ port, host: '0.0.0.0' })
} catch (err) {
  app.log.error(err)
  process.exit(1)
}