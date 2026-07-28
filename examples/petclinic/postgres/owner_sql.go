package postgres

const findOwnerSQL = `SELECT
	id,
	first_name,
	last_name,
	address,
	city,
	telephone
FROM owners
WHERE id = $1`

const countOwnersSQL = `SELECT count(*)
FROM owners
WHERE left(lower(last_name), char_length($1)) = lower($1)`

const findOwnerIDsSQL = `SELECT id
FROM owners
WHERE left(lower(last_name), char_length($1)) = lower($1)
ORDER BY lower(last_name), lower(first_name), id
OFFSET $2
LIMIT $3`

const findPetsSQL = `SELECT
	pets.id,
	pets.name,
	pets.birth_date,
	types.id,
	types.name
FROM pets
JOIN types ON types.id = pets.type_id
WHERE pets.owner_id = $1
ORDER BY lower(pets.name), pets.id`

const findVisitsSQL = `SELECT id, visit_date, description
FROM visits
WHERE pet_id = $1
ORDER BY visit_date, id`

const insertOwnerSQL = `INSERT INTO owners (
	first_name,
	last_name,
	address,
	city,
	telephone
) VALUES ($1, $2, $3, $4, $5)
RETURNING id`

const updateOwnerSQL = `UPDATE owners
SET first_name = $1,
	last_name = $2,
	address = $3,
	city = $4,
	telephone = $5
WHERE id = $6`

const insertPetSQL = `INSERT INTO pets (
	name,
	birth_date,
	type_id,
	owner_id
) VALUES ($1, $2, $3, $4)
RETURNING id`

const updatePetSQL = `UPDATE pets
SET name = $1,
	birth_date = $2,
	type_id = $3
WHERE id = $4 AND owner_id = $5`

const insertVisitSQL = `INSERT INTO visits (
	pet_id,
	visit_date,
	description
) VALUES ($1, $2, $3)
RETURNING id`

const updateVisitSQL = `UPDATE visits
SET visit_date = $1,
	description = $2
WHERE id = $3 AND pet_id = $4`
