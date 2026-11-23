CREATE TABLE IF NOT EXISTS rule_settings (
    rule_id integer PRIMARY KEY CHECK (rule_id BETWEEN 1 AND 8),
    enabled boolean NOT NULL,
    points jsonb NOT NULL CHECK (jsonb_typeof(points) = 'object'),
    parameters jsonb NOT NULL CHECK (jsonb_typeof(parameters) = 'object')
);
