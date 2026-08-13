CREATE TABLE machines (
    id UUID PRIMARY KEY,
    host TEXT NOT NULL,
    username TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE scripts (
    id UUID PRIMARY KEY,
    machine_id UUID NOT NULL,
    path TEXT NOT NULL UNIQUE,
    template TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_scripts_machine
        FOREIGN KEY (machine_id)
        REFERENCES machines(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_scripts_machine_id
    ON scripts(machine_id);

CREATE TABLE events (
    id UUID PRIMARY KEY,
    machine_id UUID,
    script_id UUID,
    username TEXT NOT NULL,
    script_path TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_events_machine
        FOREIGN KEY (machine_id)
        REFERENCES machines(id)
        ON DELETE SET NULL,

    CONSTRAINT fk_events_script
        FOREIGN KEY (script_id)
        REFERENCES scripts(id)
        ON DELETE SET NULL,

    CONSTRAINT events_action_check
        CHECK (action IN ('open', 'execute'))
);

CREATE INDEX idx_events_machine_id
    ON events(machine_id);

CREATE INDEX idx_events_script_id
    ON events(script_id);

CREATE INDEX idx_events_created_at
    ON events(created_at);
