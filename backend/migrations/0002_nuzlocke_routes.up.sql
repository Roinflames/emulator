CREATE TABLE IF NOT EXISTS nuzlocke_routes (
  id SERIAL PRIMARY KEY,
  save_id INTEGER NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
  route TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  notes TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(save_id, route)
);

CREATE INDEX IF NOT EXISTS idx_nuzlocke_routes_save_id ON nuzlocke_routes(save_id);
