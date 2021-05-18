import getpass, os, hashlib

#global variables
buyers = []

admin_buyer_dict = {   "buyer_id":"0",
                        "buyer_email":"admin@gmail.com",
                        "buyer_first_name":"admin",
                        "buyer_last_name":"buyer",
                        "buyer_password_hash":"1234"
                    }

def create_buyer_dict( buyer_id,
                        buyer_email,
                        buyer_first_name,
                        buyer_last_name,
                        buyer_password_hash
                    ):
    
    new_buyer_dict = {}
    
    new_buyer_dict["buyer_id"] = buyer_id
    new_buyer_dict["buyer_email"] = buyer_email
    new_buyer_dict["buyer_first_name"] = buyer_first_name
    new_buyer_dict["buyer_last_name"] = buyer_last_name 
    new_buyer_dict["buyer_password_hash"] = buyer_password_hash
    
    return new_buyer_dict


def load_buyer_db():
    buyer_db_handle = open("data/buyer.db","r")
    lines = buyer_db_handle.readlines()
    buyers.clear()
    count = 0
    for line in lines:
        count += 1        
        fields = line.split(",")
        #print(fields) 
        buyers.append(create_buyer_dict(  fields[0],
                                            fields[1],
                                            fields[2],
                                            fields[3],
                                            fields[4]
                                              ))
    #print(buyers)
    buyer_db_handle.close()


def buyer_init():
    if not os.path.exists("data/buyer.db"):
        buyer_db_handle = open("data/buyer.db","w")
        buyer_db_handle.close()
        save_buyer_dict(admin_buyer_dict)    
    load_buyer_db()


def save_buyer_dict(buyer_dict):
    buyer_db_handle = open("data/buyer.db","a+")
    
    output_line = str(buyer_dict["buyer_id"]+","+
                    buyer_dict["buyer_email"]+","+
                    buyer_dict["buyer_first_name"]+","+ 
                    buyer_dict["buyer_last_name"]+","+ 
                    buyer_dict["buyer_password_hash"]+"\n") 
    print(output_line)
    buyer_db_handle.write(output_line)
    buyer_db_handle.close()


def email_exists(email_to_check):
    for buyer in buyers:
        if buyer["buyer_email"] == email_to_check:
            return True
    return False

def register_buyer():
    print("<<Register Buyer>>")
    new_buyer_dict = {}
    
    new_buyer_dict['buyer_id'] = str(len(buyers))

    #TODO: check of the user exists in the buyer file    
    email=str(input("Email: "))
    while email_exists(email):
        print(email + " already exists! Please use another email")
        email=str(input("Email: "))
        
    new_buyer_dict["buyer_email"] = email
    new_buyer_dict["buyer_first_name"] = str(input("First Name: "))
    new_buyer_dict["buyer_last_name"] = str(input("Last Name: "))
    

    matched = False
    while not matched:
        password1 = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()
        password2 = hashlib.sha256(getpass.getpass("Retype Password: ").encode('utf-8')).hexdigest()
        #TODO: Make sure the password is not empty
        if password1 != password2:
            print("Password did not match! ")
        else:
            matched = True

    new_buyer_dict["buyer_password_hash"] = password1
    #print(new_buyer_dict)
    save_buyer_dict(new_buyer_dict)




    