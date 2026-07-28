package mysql

const findOwnerSQL = `SELECT
	id,
	first_name,
	last_name,
	address,
	city,
	telephone
FROM owners
WHERE id = ?`

const countOwnersSQL = `SELECT count(*)
FROM owners
WHERE left(lower(last_name), char_length(?)) = lower(?)`

const findOwnerIDsSQL = `SELECT id
FROM owners
WHERE left(lower(last_name), char_length(?)) = lower(?)
ORDER BY lower(last_name), lower(first_name), id
LIMIT ? OFFSET ?`

const findPetsSQL = `SELECT
	pets.id,
	pets.name,
	pets.birth_date,
	types.id,
	types.name
FROM pets
JOIN types ON types.id = pets.type_id
WHERE pets.owner_id = ?
ORDER BY lower(pets.name), pets.id`

const findVisitsSQL = `SELECT id, visit_date, description
FROM visits
WHERE pet_id = ?
ORDER BY visit_date, id`

const insertOwnerSQL = `INSERT INTO owners (
	first_name,
	last_name,
	address,
	city,
	telephone
) VALUES (?, ?, ?, ?, ?)`

const updateOwnerSQL = `UPDATE owners
SET first_name = ?,
	last_name = ?,
	address = ?,
	city = ?,
	telephone = ?
WHERE id = ?`

const insertPetSQL = `INSERT INTO pets (
	name,
	birth_date,
	type_id,
	owner_id
) VALUES (?, ?, ?, ?)`

const updatePetSQL = `UPDATE pets
SET name = ?,
	birth_date = ?,
	type_id = ?
WHERE id = ? AND owner_id = ?`

const insertVisitSQL = `INSERT INTO visits (
	pet_id,
	visit_date,
	description
) VALUES (?, ?, ?)`

const updateVisitSQL = `UPDATE visits
SET visit_date = ?,
	description = ?
WHERE id = ? AND pet_id = ?`
