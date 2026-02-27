module github.com/SaridakisStamatisChristos/merkle-tests

go 1.21

require (
	github.com/SaridakisStamatisChristos/vault-api v0.0.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.8.0
)

replace github.com/SaridakisStamatisChristos/vault-api => ../services/vault-api
