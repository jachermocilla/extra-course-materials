import getpass, os, hashlib

from .globals import *
from .product import *

def seller_create_dict( seller_id,
                        seller_email,
                        seller_first_name,
                        seller_last_name,
                        seller_password_hash
                    ):
    """Creates a dictionary from individual items.
    This is a helper function useful after reading 
    items from seller.db
    """

    #create an emptry dictionary
    new_seller_dict = {}

    #populate the fields
    new_seller_dict["seller_id"] = seller_id
    new_seller_dict["seller_email"] = seller_email
    new_seller_dict["seller_first_name"] = seller_first_name
    new_seller_dict["seller_last_name"] = seller_last_name 
    new_seller_dict["seller_password_hash"] = seller_password_hash
    
    #return the newly created and populated dictionary
    return new_seller_dict


def seller_load_db():
    """Reads the contents of seller.db
    The contents are placed in the sellers
    global variable defined in globals.py
    """
    #we are using the global variable
    global  sellers

    #open the file for reading
    seller_db_handle = open("data/seller.db","r")
    #read all the lines 
    lines = seller_db_handle.readlines()
    
    #remove existing items in the sellers list
    sellers.clear()

    #initialize the count to zero
    count = 0
    for line in lines:
        count += 1        
        fields = line.strip().split(",")
        
        #we call seller_create_dict() to construct
        #a dictionary for us using the fields read 
        #from the file. This dictionary is appended
        #to the global 'sellers' variable
        sellers.append(seller_create_dict(  fields[0],
                                            fields[1],
                                            fields[2],
                                            fields[3],
                                            fields[4]
                                              ))
    #close the handle 
    seller_db_handle.close()


def seller_init():
    """Initializes the seller module. 
    It creates the seller.db if it is absent 
    then calls seller_load_db().
    """

    if not os.path.exists("data/seller.db"):
        seller_db_handle = open("data/seller.db","w")
        seller_db_handle.close()
    seller_load_db()

def seller_save_dict(seller_dict):
    """Saves a dictionary of a seller to seller.db.
    """

    seller_db_handle = open("data/seller.db","a+")
    
    #construct the output line
    output_line = str(seller_dict["seller_id"]+","+
                    seller_dict["seller_email"]+","+
                    seller_dict["seller_first_name"]+","+ 
                    seller_dict["seller_last_name"]+","+ 
                    seller_dict["seller_password_hash"]+"\n") 

    #write to file then close the handle
    seller_db_handle.write(output_line)
    seller_db_handle.close()


def seller_email_exists(email_to_check):
    """Check if the email exists. 
    We don't allow duplicate emails.
    Returns True if the email exists.
    """
    global sellers

    for seller in sellers:
        if seller["seller_email"] == email_to_check:
            return True
    return False


def seller_view_register():
    """Seller registration view.
    We follow the convention of having _view_ 
    in names of functions which require user
    interaction.
    """

    print(">>[Register Seller]<<")
    
    new_seller_dict = {}
    
    #the new seller seller_id will be +1 of the last 
    new_seller_dict['seller_id'] = str(len(sellers))
    
    #make sure no duplicate email will be used 
    #since it will be used in the login
    email=str(input("Email: "))
    while seller_email_exists(email):
        print(email + " already exists! Please use another email")
        email=str(input("Email: "))

    #not a duplicate email so store in the dict    
    new_seller_dict["seller_email"] = email

    #obtain other seller information
    new_seller_dict["seller_first_name"] = str(input("First Name: "))
    new_seller_dict["seller_last_name"] = str(input("Last Name: "))
    
    #For the password, it should not be displayed so we use the 
    #getpass package. We don't store the typed password. Instead 
    #we hash it and store the hash so that it is now visible. 
    #we compare the hash instead of the actual password
    #loop until the two passwords matched.
    matched = False
    while not matched:
        password_hash_1 = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()
        password_hash_2 = hashlib.sha256(getpass.getpass("Retype Password: ").encode('utf-8')).hexdigest()
        #TODO: Make sure the password is not empty
        if password_hash_1 != password_hash_2:
            print("Password did not match! ")
        else:
            matched = True

    #store the hash of the password
    new_seller_dict["seller_password_hash"] = password_hash_1

    #save the new seller 
    seller_save_dict(new_seller_dict)

    #reload the in-memory 'sellers' variable
    seller_load_db()
    

def seller_view_login():
    """Login view
    """
    #this global variable, defined in globals.py
    #stores the currently logged-in user
    global user_session

    #email is the login name
    input_email = str(input("Email: "))

    #get and hash the password
    input_password_hash = hashlib.sha256(getpass.getpass("Password: ").encode('utf-8')).hexdigest()

    #initially set as false
    login_valid = False

    #we search for a matching email and password hash
    for seller in sellers:
        if seller["seller_email"] == input_email:
            if seller["seller_password_hash"] == input_password_hash:
                login_valid = True
                break       #exit immediately since we found it
    
    #ok we're in
    if login_valid == True:
        print("\nWelcome " + seller["seller_first_name"] + "!" )

        #we use the seller_id as the session_id
        session_id = seller["seller_id"];
        
        #we populate user_session
        user_session = {"session_id":session_id,"session_details":seller}

        #show the options for the seller
        seller_view_menu()
    else:
        input("Unknown seller! Press [ENTER] to continue.")


def seller_view_add_product():
    """Allows to seller to add a product.
    """
    global user_session

    new_product_dict = {}

    print(">>[Add Product]<<")

    new_product_dict['product_id'] = str(len(products))
    new_product_dict["product_seller_id"] = user_session["session_id"]

    #Show a guide of existing categories, populate other fields
    print("Existing Categories:>> " + str(product_get_categories().keys()))
    new_product_dict["product_category"] = str(input("Category: "))
    new_product_dict["product_name"] = str(input("Product Name: "))
    new_product_dict["product_description"] = str(input("Product Description: "))
    new_product_dict["product_quantity"] = str(input("Quantity: "))
    new_product_dict["product_unit_price"] = str(input("Unit Price: "))
    
    #save and reload by calling the functions in
    #product.py
    product_save_dict(new_product_dict)
    product_load_db()


def seller_view_menu():
    """The view for the seller options.
    """

    choice = '8'
    while choice != 'q':
        print(">>[Seller Menu]<<")
        print("[1] Add Product ")
        print("[2] Search ")
        print("[3] My Products ")
        print("[4] My Total Sales ")
        print("[q] Exit ")
        choice = str(input("Enter choice: "))
        if choice == "1":
            seller_view_add_product()
        elif choice == "2":
            #this is a function implemented in product.py
            product_view_search()   
        elif choice == "3":
            seller_view_my_products()
        elif choice == "4":
            seller_view_my_sales()


def seller_view_my_sales():
    """Compute the total sales for current logged in seller.
    """
    global sales
    global user_session

    #go over the "sales" global variable 
    # and check for items related to the current 
    # logged in user 
    total_sales = 0;
    for sale in sales:
        if sale["sale_buyer_id"] == user_session["session_id"]:
            total_sales += int(sale["sale_total_amount"])
    
    print("Your total sales: ", total_sales)
    input("Press [ENTER] to continue..")


def seller_view_my_products():
    """Show all the products of this user
    """
    global products

    print("Listed below are your products: ")
    for product in products:
        if product["product_seller_id"] == user_session["session_id"]:
            print(
                product["product_name"]+", "+
                product["product_description"]+", "+
                product["product_category"]+", "+
                product["product_unit_price"]+", "+
                product["product_quantity"]
            )
    input("Press [ENTER] to continue..")