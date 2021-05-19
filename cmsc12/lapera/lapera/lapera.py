# -*- coding: utf-8 -*-

"""lapera.lapera: provides entry point main()."""

__version__ = "0.1.0"

import io
import sys

from .seller import *
from .buyer import *
from .product import *
from .cart import *
from .sale import *

def main():
    if not os.path.exists("data"):
        os.makedirs('data')
    sale_init()
    cart_init()
    seller_init()
    buyer_init()
    product_init()
    main_menu()

def main_menu():
    choice = '8'
    while choice != 'q':
        #print("Executing lapera version %s." % __version__)
        print("Welcome to LAPERA Online Shopping!")
        print("[1] Register Seller ")
        print("[2] Register Buyer ")
        print("[3] Login Seller ")
        print("[4] Login Buyer ")
        print("[5] View 5 Random Products")
        print("[q] Exit ")
        choice = str(input("Enter choice: "))
        if choice == "1":
            seller_view_register()
        elif choice == "2":
            buyer_view_register()
        elif choice == "3":
            seller_view_login()
        elif choice == "4":
            buyer_view_login()

    print("\nThank you for using LAPERA! See you again!\n")





