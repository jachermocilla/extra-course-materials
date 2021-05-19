import getpass, os, hashlib

from .globals import *
from .product import *

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
        #seller_save_dict(seller_admin_dict)    
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

def seller_view_register():
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
    

def seller_view_login():
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
        print("\nWelcome " + seller["seller_first_name"] + "!" )
        session_id = seller["seller_id"];
        global user_session
        user_session = {"session_id":session_id,"session_details":seller}
        #print(user_session)
        seller_view_menu()
    else:
        input("Unknown seller! Press [ENTER] to continue.")

def seller_view_add_product():

    new_product_dict = {}

    print(">>Add Product<<")

    new_product_dict['product_id'] = str(len(products))
    
    global user_session
    #print(user_session)
    new_product_dict["product_seller_id"] = user_session["session_id"]
    print("Existing Categories:>> " + str(product_get_categories().keys()))
    new_product_dict["product_category"] = str(input("Category: "))
    new_product_dict["product_name"] = str(input("Product Name: "))
    new_product_dict["product_description"] = str(input("Product Description: "))
    new_product_dict["product_quantity"] = str(input("Quantity: "))
    new_product_dict["product_unit_price"] = str(input("Unit Price: "))
    
    product_save_dict(new_product_dict)
    product_load_db()

    return 0

def seller_view_menu():
    choice = '8'
    while choice != 'q':
        print(">>[Seller Menu]<<")
        print("[1] Add Product ")
        print("[2] Search ")
        print("[3] My Products ")
        print("[4] My Revenue ")
        print("[q] Exit ")
        choice = str(input("Enter choice: "))
        if choice == "1":
            seller_view_add_product()
        elif choice == "2":
            product_view_search()            
        elif choice == "3":
            seller_view_my_products()

    print("\nThank you for using LAPERA! See you again!\n")    

def seller_view_sales():
    return 0

def seller_view_my_products():
    global products

    print("Below are your products: ")
    for product in products:
        if product["product_seller_id"] == user_session["session_id"]:
            print(
                product["product_name"]+", "+
                product["product_description"]+", "+
                product["product_category"]+", "+
                product["product_unit_price"]+", "+
                product["product_quantity"]
            )



    return 0