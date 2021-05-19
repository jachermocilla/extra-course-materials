

from .globals import *
from .product import *

def cart_create_dict(   cart_id,
                        cart_buyer_id,
                        cart_product_id,
                        cart_quantity,
                        cart_checkedout,
                        cart_date
                    ):
    
    new_cart_dict = {}
    new_cart_dict["cart_id"] = cart_id
    new_cart_dict["cart_buyer_id"] = cart_buyer_id
    new_cart_dict["cart_product_id"] = cart_product_id
    new_cart_dict["cart_quantity"] = cart_quantity 
    new_cart_dict["cart_checkedout"] = cart_checkedout
    new_cart_dict["cart_date"] = cart_date

    return new_cart_dict

def cart_save_dict(cart_dict):
    cart_db_handle = open("data/cart.db","a+")
    
    output_line = str(cart_dict["cart_id"]+","+
                    cart_dict["cart_buyer_id"]+","+
                    cart_dict["cart_product_id"]+","+ 
                    cart_dict["cart_quantity"]+","+ 
                    cart_dict["cart_checkedout"]+","+
                    cart_dict["cart_date"]+"\n") 
    print(output_line)
    cart_db_handle.write(output_line)
    cart_db_handle.close()

def cart_load_db():
    cart_db_handle = open("data/cart.db","r")
    lines = cart_db_handle.readlines()
    carts.clear()
    count = 0
    for line in lines:
        count += 1        
        fields = line.strip().split(",")
        #print(fields) 
        carts.append(cart_create_dict(  fields[0],
                                            fields[1],
                                            fields[2],
                                            fields[3],
                                            fields[4],
                                            fields[5]
                                              ))
    #print(carts)
    cart_db_handle.close()


def cart_init():
    if not os.path.exists("data/cart.db"):
        cart_db_handle = open("data/cart.db","w")
        cart_db_handle.close()
    cart_load_db()


