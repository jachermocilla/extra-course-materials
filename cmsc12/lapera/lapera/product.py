import getpass, os, hashlib

products = []

def create_product_dict( product_id,
                        product_product_id,
                        product_category,
                        product_name,
                        product_description,
                        product_quantity,
                        product_unit_price
                    ):
    
    new_product_dict = {}
    
    new_product_dict["product_id"] = product_id
    new_product_dict["product_product_id"] = product_product_id
    new_product_dict["product_category"] = product_category
    new_product_dict["product_name"] = product_name
    new_product_dict["product_description"] = product_description
    new_product_dict["product_quantity"] = product_quantity
    new_product_dict["product_unit_price"] = product_unit_price
    
    return new_product_dict

def save_product_dict(product_dict):
    product_db_handle = open("data/product.db","a+")
    
    output_line = str(
                    product_dict["product_id"]+","+
                    product_dict["product_product_id"]+","+
                    product_dict["product_category"]+","+ 
                    product_dict["product_name"]+","+ 
                    product_dict["product_description"]+","+ 
                    product_dict["product_quantity"]+","+ 
                    product_dict["product_unit_price"]+"\n"                    
                    ) 
    print(output_line)
    product_db_handle.write(output_line)
    product_db_handle.close()


def load_product_db():
    product_db_handle = open("data/product.db","r")
    lines = product_db_handle.readlines()
    products.clear()
    count = 0
    for line in lines:
        count += 1        
        fields = line.strip().split(",")
        #print(fields) 
        products.append(create_product_dict(  fields[0],
                                            fields[1],
                                            fields[2],
                                            fields[3],
                                            fields[4]
                                              ))
    #print(products)
    product_db_handle.close()

def product_init():
    if not os.path.exists("data/product.db"):
        product_db_handle = open("data/product.db","w")
        product_db_handle.close()
    load_product_db()