import getpass, os, hashlib

from .lapera import *

#global variables
sellers = []

seller_admin_dict = {   "seller_id":"0",
                        "seller_email":"admin@gmail.com",
                        "seller_first_name":"admin",
                        "seller_last_name":"seller",
                        "seller_password_hash":"1234"
                    }

def seller_create_dict( seller_id,
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


def seller_load_db():
    seller_db_handle = open("data/seller.db","r")
    lines = seller_db_handle.readlines()
    sellers.clear()
    count = 0
    for line in lines:
        count += 1        
        fields = line.strip().split(",")
        #print(fields) 
        sellers.append(seller_create_dict(  fields[0],
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
        seller_save_dict(seller_admin_dict)    
    seller_load_db()


def seller_save_dict(seller_dict):
    seller_db_handle = open("data/seller.db","a+")
    
    output_line = str(seller_dict["seller_id"]+","+
                    seller_dict["seller_email"]+","+
                    seller_dict["seller_first_name"]+","+ 
                    seller_dict["seller_last_name"]+","+ 
                    seller_dict["seller_password_hash"]+"\n") 
    print(output_line)
    seller_db_handle.write(output_line)
    seller_db_handle.close()


def seller_email_exists(email_to_check):
    for seller in sellers:
        if seller["seller_email"] == email_to_check:
            return True
    return False

def seller_register():
    print("<<Register Seller>>")
    new_seller_dict = {}
    
    new_seller_dict['seller_id'] = str(len(sellers))
    
    email=str(input("Email: "))
    while seller_email_exists(email):
        print(email + " already exists! Please use another email")
        email=str(input("Email: "))
        
    new_seller_dict["seller_email"] = email
    new_seller_dict["seller_first_name"] = str(input("First Name: "))
    new_seller_dict["seller_last_name"] = str(input("Last Name: "))
    
    #TODO: check of the user exists in the seller file
    matched = False
    while not matched:
        password_hash_1 = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()
        password_hash_2 = hashlib.sha256(getpass.getpass("Retype Password: ").encode('utf-8')).hexdigest()
        #TODO: Make sure the password is not empty
        if password_hash_1 != password_hash_2:
            print("Password did not match! ")
        else:
            matched = True

    new_seller_dict["seller_password_hash"] = password_hash_1
    #print(new_seller_dict)
    seller_save_dict(new_seller_dict)
    
    #reload the in-mem sellers list
    seller_load_db()
    

def seller_login():
    input_email = str(input("Email: "))
    input_password_hash = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()

    login_valid = False
    
    for seller in sellers:
        if seller["seller_email"] == input_email:
            #print("email found")
            #print(seller["seller_password_hash"])
            #print(input_password_hash)
            if seller["seller_password_hash"] == input_password_hash:
                #print("Here")
                login_valid = True
                break
    
    if login_valid == True:
        print("Valid")
        session_id = seller["seller_id"];
        user_session = {"session_id":session_id,"session_details":seller}
        print(user_session)

    else:
        print("Invalid")



    