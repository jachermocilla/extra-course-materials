# -*- coding: utf-8 -*-


"""lapera.lapera: provides entry point main()."""

__version__ = "0.1.0"

import sys
from .register import *
from .login import *

def main():
    main_menu()

def main_menu():
    #print("Executing lapera version %s." % __version__)
    print("Welcome to LAPERA Online Shopping!")
    print("[1] Register Seller ")
    print("[2] Register Shopper ")
    print("[3] Login Seller ")
    print("[4] Login Shopper ")
    print("")
    choice = input("Enter choice: ")
    #print(choice)
    if choice == "1":
        register_seller()
    elif choice == "2":
        register_shopper()
    elif choice == "3":
        login_seller()
    else:
        login_shopper()






