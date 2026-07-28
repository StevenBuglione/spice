package mysql

var schemaSteps = []string{
	`CREATE TABLE IF NOT EXISTS vets (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		first_name VARCHAR(64) NOT NULL,
		last_name VARCHAR(64) NOT NULL,
		INDEX vets_last_name_idx (last_name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`,
	`CREATE TABLE IF NOT EXISTS specialties (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(80) NOT NULL UNIQUE,
		INDEX specialties_name_idx (name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`,
	`CREATE TABLE IF NOT EXISTS vet_specialties (
		vet_id BIGINT NOT NULL,
		specialty_id BIGINT NOT NULL,
		UNIQUE KEY unique_vet_specialty (vet_id, specialty_id),
		CONSTRAINT vet_specialties_vet_fk FOREIGN KEY (vet_id) REFERENCES vets (id),
		CONSTRAINT vet_specialties_specialty_fk FOREIGN KEY (specialty_id) REFERENCES specialties (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`,
	`CREATE TABLE IF NOT EXISTS types (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(80) NOT NULL UNIQUE,
		INDEX types_name_idx (name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`,
	`CREATE TABLE IF NOT EXISTS owners (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		first_name VARCHAR(64) NOT NULL,
		last_name VARCHAR(64) NOT NULL,
		address VARCHAR(255) NOT NULL,
		city VARCHAR(96) NOT NULL,
		telephone VARCHAR(32) NOT NULL,
		INDEX owners_last_name_idx (last_name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`,
	`CREATE TABLE IF NOT EXISTS pets (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(64) NOT NULL,
		birth_date DATE NOT NULL,
		type_id BIGINT NOT NULL,
		owner_id BIGINT NOT NULL,
		INDEX pets_name_idx (name),
		INDEX pets_owner_id_idx (owner_id),
		UNIQUE KEY unique_owner_pet_name (owner_id, name),
		CONSTRAINT pets_type_fk FOREIGN KEY (type_id) REFERENCES types (id),
		CONSTRAINT pets_owner_fk FOREIGN KEY (owner_id) REFERENCES owners (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`,
	`CREATE TABLE IF NOT EXISTS visits (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		pet_id BIGINT NOT NULL,
		visit_date DATE NOT NULL,
		description VARCHAR(255) NOT NULL,
		INDEX visits_pet_id_idx (pet_id),
		CONSTRAINT visits_pet_fk FOREIGN KEY (pet_id) REFERENCES pets (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`,
}

var seedSteps = []string{
	`INSERT IGNORE INTO vets (id, first_name, last_name) VALUES
		(1, 'James', 'Carter'), (2, 'Helen', 'Leary'),
		(3, 'Linda', 'Douglas'), (4, 'Rafael', 'Ortega'),
		(5, 'Henry', 'Stevens'), (6, 'Sharon', 'Jenkins')`,
	`INSERT IGNORE INTO specialties (id, name) VALUES
		(1, 'radiology'), (2, 'surgery'), (3, 'dentistry')`,
	`INSERT IGNORE INTO vet_specialties (vet_id, specialty_id) VALUES
		(2, 1), (3, 2), (3, 3), (4, 2), (5, 1)`,
	`INSERT IGNORE INTO types (id, name) VALUES
		(1, 'cat'), (2, 'dog'), (3, 'lizard'),
		(4, 'snake'), (5, 'bird'), (6, 'hamster')`,
	`INSERT IGNORE INTO owners
		(id, first_name, last_name, address, city, telephone) VALUES
		(1, 'George', 'Franklin', '110 W. Liberty St.', 'Madison', '6085551023'),
		(2, 'Betty', 'Davis', '638 Cardinal Ave.', 'Sun Prairie', '6085551749'),
		(3, 'Eduardo', 'Rodriquez', '2693 Commerce St.', 'McFarland', '6085558763'),
		(4, 'Harold', 'Davis', '563 Friendly St.', 'Windsor', '6085553198'),
		(5, 'Peter', 'McTavish', '2387 S. Fair Way', 'Madison', '6085552765'),
		(6, 'Jean', 'Coleman', '105 N. Lake St.', 'Monona', '6085552654'),
		(7, 'Jeff', 'Black', '1450 Oak Blvd.', 'Monona', '6085555387'),
		(8, 'Maria', 'Escobito', '345 Maple St.', 'Madison', '6085557683'),
		(9, 'David', 'Schroeder', '2749 Blackhawk Trail', 'Madison', '6085559435'),
		(10, 'Carlos', 'Estaban', '2335 Independence La.', 'Waunakee', '6085555487')`,
	`INSERT IGNORE INTO pets
		(id, name, birth_date, type_id, owner_id) VALUES
		(1, 'Leo', '2000-09-07', 1, 1), (2, 'Basil', '2002-08-06', 6, 2),
		(3, 'Rosy', '2001-04-17', 2, 3), (4, 'Jewel', '2000-03-07', 2, 3),
		(5, 'Iggy', '2000-11-30', 3, 4), (6, 'George', '2000-01-20', 4, 5),
		(7, 'Samantha', '1995-09-04', 1, 6), (8, 'Max', '1995-09-04', 1, 6),
		(9, 'Lucky', '1999-08-06', 5, 7), (10, 'Mulligan', '1997-02-24', 2, 8),
		(11, 'Freddy', '2000-03-09', 5, 9), (12, 'Lucky', '2000-06-24', 2, 10),
		(13, 'Sly', '2002-06-08', 1, 10)`,
	`INSERT IGNORE INTO visits (id, pet_id, visit_date, description) VALUES
		(1, 7, '2010-03-04', 'rabies shot'),
		(2, 8, '2011-03-04', 'rabies shot'),
		(3, 8, '2009-06-04', 'neutered'),
		(4, 7, '2008-09-04', 'spayed')`,
	`ALTER TABLE vets AUTO_INCREMENT = 7`,
	`ALTER TABLE specialties AUTO_INCREMENT = 4`,
	`ALTER TABLE types AUTO_INCREMENT = 7`,
	`ALTER TABLE owners AUTO_INCREMENT = 11`,
	`ALTER TABLE pets AUTO_INCREMENT = 14`,
	`ALTER TABLE visits AUTO_INCREMENT = 5`,
}
