CREATE TABLE contract_schedule(
	id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	contract_id INT NOT NULL,
	contract_installment_type_id INT NOT NULL,
	capital DECIMAL(13, 2) NOT NULL DEFAULT 0,
	capital_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	interest DECIMAL(13, 2) NOT NULL DEFAULT 0,
	interest_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	installment DECIMAL(13, 2) NOT NULL DEFAULT 0,
	installment_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	monthly_date DATE NOT NULL,
	daily_entry_issued BOOLEAN NOT NULL DEFAULT 0,
	marketed_capital DECIMAL(13, 2) NOT NULL DEFAULT 0,
	marketed_interest DECIMAL(13, 2) NOT NULL DEFAULT 0,
	marketed_due_date DATE NOT NULL,
	FOREIGN KEY (contract_id) REFERENCES contract(id),
	FOREIGN KEY (contract_installment_type_id) REFERENCES contract_installment_type(id)
);