import getpass

def register_seller():
    print("Register Seller")
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


