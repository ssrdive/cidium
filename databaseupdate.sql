INSERT INTO `account` (`id`, `account_category_id`, `user_id`, `datetime`, `account_id`, `name`)
VALUES
	(185, 9, 1, NOW(), 1, 'HP Stocks Out on Hire [L]'),
	(188, 24, 1, NOW(), 2, 'HP Unearned Interest Income [L]'),
	(189, 13, 1, NOW(), 3, 'HP Payable - Randeepa Agrarian Private Limited [L]'),
	(190, 19, 1, NOW(), 4, 'HP Interest Income [L]'),
	(192, 9, 1, NOW(), 5, 'HP Rentals in Arrears [L]'),
	(194, 21, 1, NOW(), 6, 'HP Interest in Suspense [L]'),
	(195, 21, 1, NOW(), 7, 'HP Bad Debt Provision [L]'),
	(196, 21, 1, NOW(), 8, 'HP Provision for Bad Debt [L]'),
	(197, 1, 1, NOW(), 75, 'Overpayments Receivables [L]'),
	(198, 1, 1, NOW(), 76, 'Recovery Charges Expenses [L]'),
	(199, 1, 1, NOW(), 77, 'Postal Charges Expenses [L]'),
	(200, 1, 1, NOW(), 78, 'Investigation Charges Expenses [L]'),
	(201, 1, 1, NOW(), 79, 'Insurance Charges Expenses [L]'),
	(202, 1, 1, NOW(), 80, 'Document Charges Income [L]'),
	(203, 1, 1, NOW(), 81, 'RMV Charges Expenses [L]'),
	(204, 1, 1, NOW(), 82, 'Default Charges Receivables [L]'),
	(205, 1, 1, NOW(), 83, 'Ceasing Charges Expenses [L]'),
	(206, 1, 1, NOW(), 84, 'Other Miscellaneous Charges Expenses [L]'),
	(207, 1, 1, NOW(), 85, 'Overpayments Income [L]'),
	(208, 1, 1, NOW(), 86, 'Recovery Charges Receivables [L]'),
	(209, 1, 1, NOW(), 87, 'Postal Charges Receivables [L]'),
	(211, 1, 1, NOW(), 88, 'Investigation Charges Receivables [L]'),
	(212, 1, 1, NOW(), 89, 'Insurance Charges Receivables [L]'),
	(213, 1, 1, NOW(), 90, 'Document Charges Receivables [L]'),
	(214, 1, 1, NOW(), 91, 'RMV Charges Receivables [L]'),
	(215, 1, 1, NOW(), 92, 'Default Charges Income [L]'),
	(216, 1, 1, NOW(), 93, 'Ceasing Charges Receivables [L]'),
	(217, 1, 1, NOW(), 94, 'Other Miscellaneous Charges Receivables [L]');

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

UPDATE contract SET lkas_17_compliant = 1 WHERE id IN (2928,2934,2939,2949,2964,2966,2971,2973,2977,2988,2992,2995,2997,3000,3002,3011,3015,3017,3020,3024,3025,3029,3032,3036,3040,3041,3042,3047,3049,3050,3051,3052,3054,3055,3056,3058,3060,3063,3065,3067,3068,3069,3072,3073,3076,3078,3080,3086,3088,3089,3090,3091,3092,3093,3094,3095,3099,2958,2959,2962,2963,2978,2979,2982,2985,2990,2993,2994,3001,3003,3004,3005,3006,3007,3014,3016,3018,3021,3022,3023,3026,3027,3028,3030,3031,3033,3034,3035,3037,3038,3039,3043,3044,3045,3048,3057,3062,3064,3066,3071,3074,3075,3079,3081,3082,3096, 3101)

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

INSERT INTO contract_payment_type VALUES (1, 'Capital'),(2, 'Interest'), (3, 'Debit');

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

UPDATE contract_installment_type SET expense_account_id = 197, receivable_account_id = 207 WHERE id = 2;
UPDATE contract_installment_type SET expense_account_id = 198, receivable_account_id = 208 WHERE id = 3;
UPDATE contract_installment_type SET expense_account_id = 199, receivable_account_id = 209 WHERE id = 4;
UPDATE contract_installment_type SET expense_account_id = 200, receivable_account_id = 211 WHERE id = 5;
UPDATE contract_installment_type SET expense_account_id = 201, receivable_account_id = 212 WHERE id = 6;
UPDATE contract_installment_type SET expense_account_id = 213, receivable_account_id = 202 WHERE id = 7;
UPDATE contract_installment_type SET expense_account_id = 203, receivable_account_id = 214 WHERE id = 8;
UPDATE contract_installment_type SET expense_account_id = 204, receivable_account_id = 215 WHERE id = 9;
UPDATE contract_installment_type SET expense_account_id = 205, receivable_account_id = 216 WHERE id = 10;
UPDATE contract_installment_type SET expense_account_id = 206, receivable_account_id = 217 WHERE id = 11;

INSERT INTO contract_financial (contract_id) VALUES 
(2958),
(2959),
(2962),
(2963),
(2978),
(2979),
(2982),
(2985),
(2990),
(2993),
(2994),
(3001),
(3003),
(3004),
(3005),
(3006),
(3007),
(3014),
(3016),
(3018),
(3021),
(3022),
(3023),
(3026),
(3027),
(3028),
(3030),
(3031),
(3033),
(3034),
(3035),
(3037),
(3038),
(3039),
(3043),
(3044),
(3045),
(3048),
(3057),
(3062),
(3064),
(3066),
(3071),
(3074),
(3075),
(3079),
(3081),
(3082),
(3096),
(2928),
(2934),
(2939),
(2949),
(2964),
(2966),
(2971),
(2973),
(2977),
(2988),
(2992),
(2995),
(2997),
(3000),
(3002),
(3011),
(3015),
(3017),
(3020),
(3024),
(3025),
(3029),
(3032),
(3036),
(3040),
(3041),
(3042),
(3047),
(3049),
(3050),
(3051),
(3052),
(3054),
(3055),
(3056),
(3058),
(3060),
(3063),
(3065),
(3067),
(3068),
(3069),
(3072),
(3073),
(3076),
(3078),
(3080),
(3086),
(3088),
(3089),
(3090),
(3091),
(3092),
(3093),
(3094),
(3095),
(3099),
(3101);