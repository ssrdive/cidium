import sys
import logging
import requests
import optparse
import mysql.connector
import ConfigParser

API_URL = "https://agrivest.app/api-dev"

def setup_logger():
    formatter = logging.Formatter("%(asctime)s : %(funcName)s : %(levelname)s : %(message)s")
    ch = logging.StreamHandler()
    ch.setFormatter(formatter)
    
    global log
    log = logging.getLogger("issue_charges")
    log.addHandler(ch)
    log.setLevel(logging.DEBUG)

def parse_arguments():
    log.info("Parsing arguments and auth.config file")
    parser = optparse.OptionParser()
    parser.add_option('--contract', '-c', default="0")
    parser.add_option('--doc_charges', '-d', default="0")
    parser.add_option('--ins_charges', '-i', default="0")
    parser.add_option('--receipt', '-r', default="0")
    global options
    options, arguments = parser.parse_args()

    config = ConfigParser.RawConfigParser()
    config.read('auth.config')
    global client_username, client_passowrd
    global db_host, db_username, db_password, db_instance
    client_username = config.get('CLIENT_CREDENTIALS', 'USERNAME')
    client_passowrd = config.get('CLIENT_CREDENTIALS', 'PASSWORD')
    db_host = config.get('DB_CREDENTIALS', 'HOST')
    db_username = config.get('DB_CREDENTIALS', 'USERNAME')
    db_password = config.get('DB_CREDENTIALS', 'PASSWORD')
    db_instance = config.get('DB_CREDENTIALS', 'INSTANCE')

def validate_response(res):
    if res.status_code == 200:
        return res.json()
    else:
        log.error("Request returned " + str(res.status_code))
        sys.exit(-1)

def authenticate():
    log.info("Authenticating")
    r = requests.post(url = API_URL + "/authenticate", data = {
        'username': client_username,
        'password': client_passowrd
    })

    global user
    user = validate_response(r)

def connect_to_db():
    log.info("Connecting to database")
    global db
    db = mysql.connector.connect(
        host = db_host,
        user = db_username,
        password = db_password,
        database = db_instance
    )

def send_post(url, data):
    log.info(("Sending POST request to %s" % url))
    r = requests.post(url = API_URL + url, headers = {
        "Authorization": "Bearer " + user["token"]
    }, data = data)
    return validate_response(r)


def issue_charges():
    log.info(("Issuing charges for contract %s doc charges - %s insurance charges %s" % (options.contract, options.doc_charges, options.ins_charges)))

    if float(options.doc_charges) + float(options.ins_charges) != float(options.receipt):
        log.error("Document charnges and insurance charnges does not tally with receipt amount")
        sys.exit(-1)

    cursor = db.cursor()
    cursor.execute("SELECT customer_contact FROM contract WHERE id = " + options.contract)
    results = cursor.fetchone()
    current_mobile = results[0]
    log.info(("Current mobile: %s" % current_mobile))
    cursor.execute("UPDATE contract SET customer_contact = 768237192 WHERE id = " + options.contract)
    db.commit()
    
    if float(options.doc_charges) > 0:
        send_post("/contract/debitnote", {
            'contract_id': options.contract,
            'capital': options.doc_charges,
            'contract_installment_type_id': 7,
            'notes': '[AUTOMATED DEBIT]',
            'user_id': user['id']
        })

    if float(options.ins_charges) > 0:
        send_post("/contract/debitnote", {
            'contract_id': options.contract,
            'capital': options.ins_charges,
            'contract_installment_type_id': 6,
            'notes': '[AUTOMATED DEBIT]',
            'user_id': user['id']
        })

    if float(options.doc_charges) + float(options.ins_charges) == float(options.receipt):
        send_post("/contract/receipt", {
            'cid': options.contract,
            'amount': options.receipt,
            'notes': '[AUTOMATED RECEIPT]',
            'user_id': user['id']
        })

    cursor.execute("UPDATE contract SET customer_contact = " + str(current_mobile) + " WHERE id = " + options.contract)
    db.commit()
    cursor.close()
    db.close()

    log.info("Charges issued successfully")

def main():
    setup_logger()
    parse_arguments()
    authenticate()
    connect_to_db()
    issue_charges()

if __name__ == '__main__':
    main()