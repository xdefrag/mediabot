-- +goose Up
-- +goose StatementBegin
CREATE TABLE responses (
    from_user_id bigint NOT NULL,
    to_user_id bigint NOT NULL,
    message text NOT NULL,
    created_at timestamp with time zone NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE responses;
-- +goose StatementEnd
