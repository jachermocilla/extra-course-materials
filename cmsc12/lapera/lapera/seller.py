import getpass, os, hashlib

#global variables
sellers = []

admin_seller_dict = {   "seller_id":"0",
                        "seller_email":"admin@gmail.com",
                        "seller_first_name":"admin",
                        "seller_last_name":"seller",
                        "seller_password_hash":"1234"
                    }

def create_seller_dict( seller_id,
                        seller_email,
                        seller_first_name,
                        seller_last_name,
                        seller_password_hash
                    ):
    
    new_seller_dict = {}
    
    new_seller_dict["seller_id"] = seller_id
    new_seller_dict["seller_email"] = seller_email
    new_seller_dict["seller_first_name"] = seller_first_name
    new_seller_dict["seller_last_name"] = seller_last_name 
    new_seller_dict["seller_password_hash"] = seller_password_hash
    
    return new_seller_dict


def load_seller_db():
    seller_db_handle = open("data/seller.db","r")
    lines = seller_db_handle.readlines()
    sellers.clear()
    count = 0
    for line in lines:
        count += 1        
        fields = line.split(",")
        #print(fields) 
        sellers.append(create_seller_dict(  fields[0],
                                            fields[1],
                                            fields[2],
                                            fields[3],
                                            fields[4]
                                              ))
    #print(sellers)
    seller_db_handle.close()


def seller_init():
    if not os.path.exists("data/seller.db"):
        seller_db_handle = open("data/seller.db","w")
        seller_db_handle.close()
        save_seller_dict(admin_seller_dict)    
    load_seller_db()


def save_seller_dict(seller_dict):
    seller_db_handle = open("data/seller.db","a+")
    
    output_line = str(seller_dict["seller_id"]+","+
                    seller_dict["seller_email"]+","+
                    seller_dict["seller_first_name"]+","+ 
                    seller_dict["seller_last_name"]+","+ 
                    seller_dict["seller_password_hash"]+"\n") 
    print(output_line)
    seller_db_handle.write(output_line)
    seller_db_handle.close()


def email_exists(email_to_check):
    for seller in sellers:
        if seller["seller_email"] == email_to_check:
            return True
    return False

def register_seller():
    print("<<Register Seller>>")
    new_seller_dict = {}
    
    new_seller_dict['seller_id'] = str(len(sellers))
    
    email=str(input("Email: "))
    while email_exists(email):
        print(email + " already exists! Please use another email")
        email=str(input("Email: "))
        
    new_seller_dict["seller_email"] = email
    new_seller_dict["seller_first_name"] = str(input("First Name: "))
    new_seller_dict["seller_last_name"] = str(input("Last Name: "))
    
    #TODO: check of the user exists in the seller file
    matched = False
    while not matched:
        password1 = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()
        password2 = hashlib.sha256(getpass.getpass("Retype Password: ").encode('utf-8')).hexdigest()
        #TODO: Make sure the password is not empty
        if password1 != password2:
            print("Password did not match! ")
        else:
            matched = True

    new_seller_dict["seller_password_hash"] = password1
    #print(new_seller_dict)
    save_seller_dict(new_seller_dict)




    