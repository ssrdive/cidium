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
	marketed_installment BOOLEAN NOT NULL DEFAULT 0,
	marketed_capital DECIMAL(13, 2) NOT NULL DEFAULT 0,
	marketed_capital_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	marketed_interest DECIMAL(13, 2) NOT NULL DEFAULT 0,
	marketed_interest_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	marketed_due_date DATE NOT NULL,
	FOREIGN KEY (contract_id) REFERENCES contract(id),
	FOREIGN KEY (contract_installment_type_id) REFERENCES contract_installment_type(id)
);

CREATE TABLE contract_schedule_charges_debits_details(
	contract_schedule_id INT NOT NULL,
	user_id INT NOT NULL,
	notes VARCHAR(2048),
	FOREIGN KEY (contract_schedule_id) REFERENCES contract_schedule(id),
	FOREIGN KEY (user_id) REFERENCES user(id)
);

ALTER TABLE contract
ADD lkas_17_compliant BOOLEAN NOT NULL DEFAULT 1 AFTER id;

UPDATE contract SET lkas_17_compliant = 0;

CREATE TABLE recovery_status(
	id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(32) NOT NULL
);

INSERT INTO recovery_status VALUES (1, 'Active'), (2, 'Arrears'), (3, 'Non-performing Loan'), (4, 'Bad Debt Provisioned');

CREATE TABLE contract_financial(
	id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	contract_id INT NOT NULL,
	active BOOLEAN NOT NULL DEFAULT 1,
	recovery_status_id INT NOT NULL DEFAULT 1,
	doubtful BOOLEAN NOT NULL DEFAULT 0,
	payment DECIMAL(13, 2) NOT NULL DEFAULT 0,
	agreed_capital DECIMAL(13, 2) NOT NULL DEFAULT 0,
	agreed_interest DECIMAL(13, 2) NOT NULL DEFAULT 0,
	capital_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	interest_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	charges_debits_paid DECIMAL(13, 2) NOT NULL DEFAULT 0,
	capital_arrears DECIMAL(13, 2) NOT NULL DEFAULT 0,
	interest_arrears DECIMAL(13, 2) NOT NULL DEFAULT 0,
	charges_debits_arrears DECIMAL(13, 2) NOT NULL DEFAULT 0,
	capital_provisioned DECIMAL(13, 2) NOT NULL DEFAULT 0,
	financial_schedule_start_date DATE,
	financial_schedule_end_date DATE,
	marketed_schedule_start_date DATE,
	marketed_schedule_end_date DATE,
	payment_interval INT NOT NULL DEFAULT 0,
	payments INT NOT NULL DEFAULT 0,
	FOREIGN KEY (contract_id) REFERENCES contract(id),
	FOREIGN KEY (recovery_status_id) REFERENCES recovery_status(id)
);

ALTER TABLE contract_receipt
ADD lkas_17 BOOLEAN NOT NULL DEFAULT 0 AFTER id;

/* Should update the new contracts' lkas_17_compliant
   column to reflect they are compliant
   before initiating */

CREATE TABLE contract_payment_type(
	id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(32) NOT NULL
);

INSERT INTO contract_payment_type VALUES (1, 'Capital'),(2, 'Interest');

CREATE TABLE contract_financial_payment(
	id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	contract_payment_type_id INT NOT NULL,
	contract_schedule_id INT NOT NULL,
	contract_receipt_id INT NOT NULL,
	amount DECIMAL(13, 2) NOT NULL DEFAULT 0,
	FOREIGN KEY (contract_payment_type_id) REFERENCES contract_payment_type(id),
	FOREIGN KEY (contract_schedule_id) REFERENCES contract_schedule(id),
	FOREIGN KEY (contract_receipt_id) REFERENCES contract_receipt(id)
);

CREATE TABLE contract_marketed_payment(
	id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	contract_payment_type_id INT NOT NULL,
	contract_schedule_id INT NOT NULL,
	contract_receipt_id INT NOT NULL,
	amount DECIMAL(13, 2) NOT NULL DEFAULT 0,
	FOREIGN KEY (contract_payment_type_id) REFERENCES contract_payment_type(id)
);

ALTER TABLE contract_installment_type
ADD expense_account_id INT,
ADD receivable_account_id INT,
ADD FOREIGN KEY (expense_account_id) REFERENCES account(id),
ADD FOREIGN KEY (receivable_account_id) REFERENCES account(id)