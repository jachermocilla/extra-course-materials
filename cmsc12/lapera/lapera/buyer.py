import getpass, os, hashlib
from datetime import datetime

from .globals import *
from .product import *
from .cart import *
from .sale import *


#global variables


#buyer_admin_dict = {   "buyer_id":"0",
#                        "buyer_email":"admin@gmail.com",
#                        "buyer_first_name":"admin",
#                        "buyer_last_name":"buyer",
#                        "buyer_password_hash":"1234"
#                    }

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
        #buyer_save_dict(buyer_admin_dict)    
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
        global user_session
        print("\nWelcome " + buyer["buyer_first_name"] + "!" )
        session_id = buyer["buyer_id"];
        user_session = {"session_id":session_id,"session_details":buyer}
        #print(user_session)
        buyer_view_menu()
    else:
        input("Unknown buyer! Press [ENTER] to continue.")

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
        print(">>[Buyer Menu]<<")
        print("[1] Search/Add To Cart ")
        print("[2] View Cart")        
        print("[3] View Total Expenses")        
        print("[q] Exit ")
        choice = str(input("Enter choice: "))
        if choice == "1":
            buyer_view_search()
        elif choice == "2":
            buyer_view_cart()
        elif choice == "3":
            buyer_view_total_expenses()
        
    print("\nThank you for using LAPERA! See you again!\n")

def buyer_view_cart():
    global carts
    global user_session
    global products
    global sales
    
    print("Below are the contents of your cart:")
    
    current_user_cart=[]
    for cart_item in carts:
        if cart_item["cart_buyer_id"] == user_session["session_id"]:
            if cart_item["cart_checkedout"] != "1":
                current_user_cart.append(cart_item)

    count = 0
    for cart_item in current_user_cart:
        product = products[int(cart_item["cart_product_id"])]
        print(  "[" + cart_item["cart_id"] + "] - " +
                product["product_name"]+" - " 
                + cart_item["cart_quantity"] + " units " )
        count = count + 1

    print("There are " +  str(len(current_user_cart)) + " item(s)")
    checkout = str(input("Checkout an item?[y/n]"))
    if checkout == 'y':
        cart_item_id = str(input("Enter [item id] of item: "))
        
        for cart_item in current_user_cart:
            if cart_item["cart_id"] == cart_item_id:
                break;
        
        #print(cart_item)
        product = products[int(cart_item["cart_product_id"])]
        
        total_amount = int(cart_item["cart_quantity"]) * \
                            int(product["product_unit_price"])
        #print(str(total_amount))
        
        remaining_qty = int(product["product_quantity"]) - \
                            int(cart_item["cart_quantity"])

        #print(remaining_qty)

        sale_save_dict(sale_create_dict(str(len(sales)),
                            cart_item["cart_buyer_id"],
                            cart_item["cart_product_id"],
                            cart_item["cart_quantity"],
                            str(total_amount),
                            str(datetime.now())
                        ))
        
        #reload sale db
        sale_load_db()

        #update the quantity in products
        product["product_quantity"] = str(remaining_qty)

        #make the cart item checkedout
        cart_item["cart_checkedout"] = "1"
        #print(carts)
        #print(products)
        
        #flush to file and reload
        product_flush_to_file()
        #product_load_db()
        cart_flush_to_file()
        #cart_load_db()
        input("The item will be delivered within 5 days. Please press [ENTER]..")


def buyer_view_total_expenses():
    global sales
    global user_session

    total_expenses = 0;
    for sale in sales:
        if sale["sale_buyer_id"] == user_session["session_id"]:
            total_expenses += int(sale["sale_total_amount"])
    print("Your total expenses: ", total_expenses)

def buyer_view_search():
    product_view_search()
    add_to_cart = str(input("Add to cart?[y/n]"))

    global user_session
    global carts
    if add_to_cart == 'y':
        product_id = str(input("Enter product id of item: "))
        qty = str(input("How many units of the product? "))
        cart_save_dict(cart_create_dict(   str(len(carts)),
                            user_session["session_id"],
                            product_id,
                            qty,
                            "0",
                            str(datetime.now()))
                    )
        cart_load_db()




