from .globals import *
from .product import *

def sale_create_dict(   sale_id,
                        sale_buyer_id,
                        sale_product_id,
                        sale_quantity,
                        sale_total_amount,
                        sale_date
                    ):
    
    new_sale_dict = {}
    new_sale_dict["sale_id"] = sale_id
    new_sale_dict["sale_buyer_id"] = sale_buyer_id
    new_sale_dict["sale_product_id"] = sale_product_id
    new_sale_dict["sale_quantity"] = sale_quantity 
    new_sale_dict["sale_total_amount"] = sale_total_amount
    new_sale_dict["sale_date"] = sale_date

    return new_sale_dict

def sale_save_dict(sale_dict):
    sale_db_handle = open("data/sale.db","a+")
    
    output_line = str(sale_dict["sale_id"]+","+
                    sale_dict["sale_buyer_id"]+","+
                    sale_dict["sale_product_id"]+","+ 
                    sale_dict["sale_quantity"]+","+ 
                    sale_dict["sale_total_amount"]+","+
                    sale_dict["sale_date"]+"\n") 
    #print(output_line)
    sale_db_handle.write(output_line)
    sale_db_handle.close()

def sale_load_db():
    sale_db_handle = open("data/sale.db","r")
    lines = sale_db_handle.readlines()
    sales.clear()
    count = 0
    for line in lines:
        count += 1        
        fields = line.strip().split(",")
        #print(fields) 
        sales.append(sale_create_dict(  fields[0],
                                            fields[1],
                                            fields[2],
                                            fields[3],
                                            fields[4],
                                            fields[5]
                                              ))
    #print(sales)
    sale_db_handle.close()


def sale_init():
    if not os.path.exists("data/sale.db"):
        sale_db_handle = open("data/sale.db","w")
        sale_db_handle.close()
    sale_load_db()


