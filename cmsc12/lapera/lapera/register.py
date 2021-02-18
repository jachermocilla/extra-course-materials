import getpass

def register_seller():
    print("Register Seller")
    username1 = input("Username: ")
    #TODO: check of the user exists in the seller file
    #

    password1 = getpass.getpass("Password: ")
    password2 = getpass.getpass("Retype Password: ")
    if password1 != password2:
        print("Password did not match! ")
    else:
        print("New Seller Added! ")
        main_menu()

def register_shopper():
    print("Register Shopper")


