import getpass, os, hashlib

from .globals import *
from .product import *


#global variables


buyer_admin_dict = {   "buyer_id":"0",
                        "buyer_email":"admin@gmail.com",
                        "buyer_first_name":"admin",
                        "buyer_last_name":"buyer",
                        "buyer_password_hash":"1234"
                    }

def buyer_create_dict( buyer_id,
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


def buyer_load_db():
    buyer_db_handle = open("data/buyer.db","r")
    lines = buyer_db_handle.readlines()
    buyers.clear()
    count = 0
    for line in lines:
        count += 1        
        fields = line.strip().split(",")
        #print(fields) 
        buyers.append(buyer_create_dict(  fields[0],
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
        buyer_save_dict(buyer_admin_dict)    
    buyer_load_db()


def buyer_save_dict(buyer_dict):
    buyer_db_handle = open("data/buyer.db","a+")
    
    output_line = str(buyer_dict["buyer_id"]+","+
                    buyer_dict["buyer_email"]+","+
                    buyer_dict["buyer_first_name"]+","+ 
                    buyer_dict["buyer_last_name"]+","+ 
                    buyer_dict["buyer_password_hash"]+"\n") 
    print(output_line)
    buyer_db_handle.write(output_line)
    buyer_db_handle.close()


def buyer_email_exists(email_to_check):
    for buyer in buyers:
        if buyer["buyer_email"] == email_to_check:
            return True
    return False

def buyer_view_register():
    print("<<Register buyer>>")
    new_buyer_dict = {}
    
    new_buyer_dict['buyer_id'] = str(len(buyers))
    
    email=str(input("Email: "))
    while buyer_email_exists(email):
        print(email + " already exists! Please use another email")
        email=str(input("Email: "))
        
    new_buyer_dict["buyer_email"] = email
    new_buyer_dict["buyer_first_name"] = str(input("First Name: "))
    new_buyer_dict["buyer_last_name"] = str(input("Last Name: "))
    
    #TODO: check of the user exists in the buyer file
    matched = False
    while not matched:
        password_hash_1 = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()
        password_hash_2 = hashlib.sha256(getpass.getpass("Retype Password: ").encode('utf-8')).hexdigest()
        #TODO: Make sure the password is not empty
        if password_hash_1 != password_hash_2:
            print("Password did not match! ")
        else:
            matched = True

    new_buyer_dict["buyer_password_hash"] = password_hash_1
    #print(new_buyer_dict)
    buyer_save_dict(new_buyer_dict)
    
    #reload the in-mem buyers list
    buyer_load_db()
    

def buyer_view_login():
    input_email = str(input("Email: "))
    input_password_hash = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()

    login_valid = False
    
    for buyer in buyers:
        if buyer["buyer_email"] == input_email:
            #print("email found")
            #print(buyer["buyer_password_hash"])
            #print(input_password_hash)
            if buyer["buyer_password_hash"] == input_password_hash:
                #print("Here")
                login_valid = True
                break
    
    if login_valid == True:
        print("Valid")
        session_id = buyer["buyer_id"];
        global user_session
        user_session = {"session_id":session_id,"session_details":buyer}
        #print(user_session)
        buyer_view_menu()
    else:
        print("Invalid")

def buyer_view_add_product():

    new_product_dict = {}

    print(">>Add Product<<")

    new_product_dict['product_id'] = str(len(products))
    
    global user_session
    #print(user_session)
    new_product_dict["product_buyer_id"] = user_session["session_id"]
    
    new_product_dict["product_category"] = str(input("Category: "))
    new_product_dict["product_name"] = str(input("Product Name: "))
    new_product_dict["product_description"] = str(input("Product Description: "))
    new_product_dict["product_quantity"] = str(input("Quantity: "))
    new_product_dict["product_unit_price"] = str(input("Unit Price: "))
    
    product_save_dict(new_product_dict)
    product_load_db()

    return 0

def buyer_view_menu():
    choice = '8'
    while choice != 'q':
        print(">>[buyer Menu]<<")
        print("[1] Add Product ")
        print("[2] View Sales ")
        print("[q] Exit ")
        choice = str(input("Enter choice: "))
        if choice == "1":
            buyer_view_add_product()
        elif choice == "2":
            buyer_view_register()
        elif choice == "3":
            buyer_view_login()
        elif choice == "4":
            buyer_view_login()
    print("\nThank you for using LAPERA! See you again!\n")