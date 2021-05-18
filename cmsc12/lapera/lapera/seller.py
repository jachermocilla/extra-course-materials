import getpass, os

#the global list of sellers
sellers = {}
seller_next_id = None
seller_db_handle = None

root_seller_dict = {    "seller_id":"0",
                        "seller_first_name":"root",
                        "seller_last_name":"seller",
                        "seller_login_name":"root_seller",
                        "seller_password_hash":"1234"}


def create_seller_dict( seller_id,
                        seller_first_name,
                        seller_last_name,
                        seller_login_name,
                        seller_password_hash
                    ):
    
    new_seller_dict = {}
    
    new_seller_dict["seller_id"] = seller_id
    new_seller_dict["seller_first_name"] = seller_first_name
    new_seller_dict["seller_last_name"] = seller_last_name 
    new_seller_dict["seller_login_name"] = seller_login_name 
    new_seller_dict["seller_password_hash"] = seller_password_hash
    
    return new_seller_dict


def seller_init():
    if not os.path.exists("data/seller.db"):
        seller_db_handle = open("data/seller.db","w")
        seller_db_handle.close()
    
    save_seller_dict(root_seller_dict)    

    
def save_seller_dict(seller_dict):
    seller_db_handle = open("data/seller.db","r+")
    
    output_line = seller_dict["seller_id"]+","+
                    seller_dict["seller_first_name"]+","+
                    seller_dict["seller_last_name"]+","+
                    seller_dict["seller_login_name"]+","+
                    seller_dict["seller_password_hash"]
    seller_db_handle.write(output_line)
    seller_db_handle.close()

def load_seller():
    seller_db_handle = open("data/seller.db","r")

def register_seller():
    print("<<Register Seller>>")


    newusername = str(input("Username: "))
    #TODO: check of the user exists in the seller file
    seller_db = []
    seller_db_handle = open("data/seller.db","r")
    for line in seller_db_handle:
        seller = line[:-1]
        seller_db.append(seller)
    
    seller_db_handle.close()

    print(seller_db)

    password1 = getpass.getpass("Password: ")
    password2 = getpass.getpass("Retype Password: ")
    #TODO: Make sure the password is not empty
    if password1 != password2:
        print("Password did not match! ")
    else:
        print("New Seller Added! ")

def register_shopper():
    print("Register Shopper")



    