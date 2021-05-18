# -*- coding: utf-8 -*-


"""lapera.lapera: provides entry point main()."""

__version__ = "0.1.0"


import io
import sys

from .register import *
from .login import *

from .seller import *
from .buyer import *

def main():
    seller_init()
    buyer_init()
    main_menu()
    #register_seller()

def main_menu():
    #print("Executing lapera version %s." % __version__)
    print("Welcome to LAPERA Online Shopping!")
    print("[1] Register Seller ")
    print("[2] Register Shopper ")
    print("[3] Login Seller ")
    print("[4] Login Shopper ")
    choice = str(input("Enter choice: "))
    #print(choice)
    if choice == "1":
        register_seller()
    elif choice == "2":
        register_buyer()
    elif choice == "3":
        login_seller()
    elif choice == "4":
        login_shopper()
    else:
        print("Unknown choice",choice,"Exiting.")






