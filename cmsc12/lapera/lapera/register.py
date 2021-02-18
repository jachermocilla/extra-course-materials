import getpass

def register_seller():
    print("Register Seller")
    username1 = input("Username: ")
    password1 = getpass.getpass("Password: ")
    password2 = getpass.getpass("Retype Password: ")
    if password1 != password2:
        print("Password did not match!")

def register_shopper():
    print("Register Shopper")


