import "dotenv/config";

import { Pool } from 'pg'


const pool = new Pool({
  user: process.env.DB_USER ?? process.env.DBUSER,
  host: process.env.DB_HOST ?? process.env.HOST,
  database: process.env.DB_NAME ?? process.env.DATABASE,
  password: process.env.DB_PASS ?? process.env.PASSWORD,
  port: Number.parseInt(process.env.DB_PORT ?? process.env.PORT ?? '5432', 10),
  max: Number.parseInt(process.env.DB_POOL_MAX ?? '10', 10),
});

export async function buscarEventos() {
  const res = await pool.query('SELECT id, nome, ingressos_disponiveis FROM eventos');
  return res.rows;
}

export async function reservarIngresso(eventoId, usuarioId) {
  const res = await pool.query({
  name: 'reserve-ticket',
  text: `
    WITH updated AS (
  UPDATE eventos
  SET ingressos_disponiveis = ingressos_disponiveis - 1
  WHERE id = $1 AND ingressos_disponiveis > 0
  RETURNING id
)
INSERT INTO reservas (evento_id, usuario_id)
SELECT id, $2 FROM updated;
  `,
  values: [eventoId, usuarioId]
});

  return res.rowCount > 0;
}