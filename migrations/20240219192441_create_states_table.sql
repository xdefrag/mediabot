-- +goose Up
-- +goose StatementBegin
CREATE TABLE states (
    user_id bigint NOT NULL,
    state text NOT NULL,
    data json NOT NULL,
    meta json NOT NULL,
    created_at timestamp with time zone NOT NULL
);

CREATE INDEX idx_states_user_id_created_at_desc
ON states (user_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_states_user_id_created_at_desc;
DROP TABLE states;
-- +goose StatementEnd
