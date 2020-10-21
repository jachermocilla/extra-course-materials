-- Copyright (C) 1991-2013 Altera Corporation
-- Your use of Altera Corporation's design tools, logic functions 
-- and other software and tools, and its AMPP partner logic 
-- functions, and any output files from any of the foregoing 
-- (including device programming or simulation files), and any 
-- associated documentation or information are expressly subject 
-- to the terms and conditions of the Altera Program License 
-- Subscription Agreement, Altera MegaCore Function License 
-- Agreement, or other applicable license agreement, including, 
-- without limitation, that your use is for the sole purpose of 
-- programming logic devices manufactured by Altera and sold by 
-- Altera or its authorized distributors.  Please refer to the 
-- applicable agreement for further details.

-- PROGRAM		"Quartus II 64-Bit"
-- VERSION		"Version 13.1.0 Build 162 10/23/2013 SJ Web Edition"
-- CREATED		"Fri May 11 02:04:52 2018"

LIBRARY ieee;
USE ieee.std_logic_1164.all; 

LIBRARY work;

ENTITY Processor IS 
	PORT
	(
		clk :  IN  STD_LOGIC;
		current_instruction :  OUT  STD_LOGIC_VECTOR(2 DOWNTO 0);
		value :  OUT  STD_LOGIC_VECTOR(7 DOWNTO 0)
	);
END Processor;

ARCHITECTURE bdf_type OF Processor IS 

COMPONENT alu
	PORT(op : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 rs : IN STD_LOGIC_VECTOR(7 DOWNTO 0);
		 rt : IN STD_LOGIC_VECTOR(7 DOWNTO 0);
		 rd : OUT STD_LOGIC_VECTOR(7 DOWNTO 0)
	);
END COMPONENT;

COMPONENT control
	PORT(instr : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 alu_src : OUT STD_LOGIC;
		 reg_dst : OUT STD_LOGIC;
		 alu_op : OUT STD_LOGIC_VECTOR(1 DOWNTO 0)
	);
END COMPONENT;

COMPONENT instruction
	PORT(instr_addr : IN STD_LOGIC_VECTOR(2 DOWNTO 0);
		 op : OUT STD_LOGIC_VECTOR(1 DOWNTO 0);
		 rd_addr : OUT STD_LOGIC_VECTOR(1 DOWNTO 0);
		 rs_addr : OUT STD_LOGIC_VECTOR(1 DOWNTO 0);
		 rt_addr : OUT STD_LOGIC_VECTOR(1 DOWNTO 0)
	);
END COMPONENT;

COMPONENT mux0
	PORT(sel : IN STD_LOGIC;
		 a : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 b : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 y : OUT STD_LOGIC_VECTOR(1 DOWNTO 0)
	);
END COMPONENT;

COMPONENT mux1
	PORT(sel : IN STD_LOGIC;
		 a : IN STD_LOGIC_VECTOR(7 DOWNTO 0);
		 b : IN STD_LOGIC_VECTOR(7 DOWNTO 0);
		 y : OUT STD_LOGIC_VECTOR(7 DOWNTO 0)
	);
END COMPONENT;

COMPONENT pc
	PORT(clk : IN STD_LOGIC;
		 current_instr : IN STD_LOGIC_VECTOR(2 DOWNTO 0);
		 next_instr : OUT STD_LOGIC_VECTOR(2 DOWNTO 0)
	);
   END COMPONENT;

COMPONENT registers
	PORT(clk : IN STD_LOGIC;
		 rd_addr : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 rs_addr : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 rt_addr : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 wr_data : IN STD_LOGIC_VECTOR(7 DOWNTO 0);
		 rs : OUT STD_LOGIC_VECTOR(7 DOWNTO 0);
		 rt : OUT STD_LOGIC_VECTOR(7 DOWNTO 0)
	);
END COMPONENT;

COMPONENT sign_extend
	PORT(data_in : IN STD_LOGIC_VECTOR(1 DOWNTO 0);
		 data_out : OUT STD_LOGIC_VECTOR(7 DOWNTO 0)
	);
END COMPONENT;

SIGNAL	SYNTHESIZED_WIRE_0 :  STD_LOGIC_VECTOR(1 DOWNTO 0);      --op (alu), alu_op (control)
SIGNAL	SYNTHESIZED_WIRE_1 :  STD_LOGIC_VECTOR(7 DOWNTO 0);      --rs (alu)
SIGNAL	SYNTHESIZED_WIRE_2 :  STD_LOGIC_VECTOR(7 DOWNTO 0);      --rt (alu)
SIGNAL	SYNTHESIZED_WIRE_3 :  STD_LOGIC_VECTOR(1 DOWNTO 0);      --instr (control), op (instruciton) 
SIGNAL	SYNTHESIZED_WIRE_17 :  STD_LOGIC_VECTOR(2 DOWNTO 0);     --current_instruction, instr_addr (instruction), current_instr, next_instr (pc)
SIGNAL	SYNTHESIZED_WIRE_5 :  STD_LOGIC;                         --reg_dst (control), sel (mux0) 
SIGNAL	SYNTHESIZED_WIRE_18 :  STD_LOGIC_VECTOR(1 DOWNTO 0);     --rt_addr (instruction), a (mux0), rt_addr (registers)
SIGNAL	SYNTHESIZED_WIRE_19 :  STD_LOGIC_VECTOR(1 DOWNTO 0);     --rd_addr (instruction), b (mux0), data_in (sign_extend)
SIGNAL	SYNTHESIZED_WIRE_8 :  STD_LOGIC;                         --alu_src (control), sel (mux1)
SIGNAL	SYNTHESIZED_WIRE_9 :  STD_LOGIC_VECTOR(7 DOWNTO 0);      --a (mux1), rt (registers)
SIGNAL	SYNTHESIZED_WIRE_10 :  STD_LOGIC_VECTOR(7 DOWNTO 0);     --b (mux1), data_out (sign_extend)
SIGNAL	SYNTHESIZED_WIRE_12 :  STD_LOGIC_VECTOR(1 DOWNTO 0);     --y (mux0), rd_addr(registers)
SIGNAL	SYNTHESIZED_WIRE_13 :  STD_LOGIC_VECTOR(1 DOWNTO 0);     --rs_addr(instruction), rs_addr(registers)
SIGNAL	SYNTHESIZED_WIRE_15 :  STD_LOGIC_VECTOR(7 DOWNTO 0);     --value (Processor),rd (alu), wr_data (registers)


BEGIN 
current_instruction <= SYNTHESIZED_WIRE_17;
value <= SYNTHESIZED_WIRE_15;



b2v_arithmetic_logic_unit : alu
PORT MAP(op => SYNTHESIZED_WIRE_0,
		 rs => SYNTHESIZED_WIRE_1,
		 rt => SYNTHESIZED_WIRE_2,
		 rd => SYNTHESIZED_WIRE_15);


b2v_control_unit : control
PORT MAP(instr => SYNTHESIZED_WIRE_3,
		 alu_src => SYNTHESIZED_WIRE_8,
		 reg_dst => SYNTHESIZED_WIRE_5,
		 alu_op => SYNTHESIZED_WIRE_0);


b2v_instruction_memory : instruction
PORT MAP(instr_addr => SYNTHESIZED_WIRE_17,
		 op => SYNTHESIZED_WIRE_3,
		 rd_addr => SYNTHESIZED_WIRE_19,
		 rs_addr => SYNTHESIZED_WIRE_13,
		 rt_addr => SYNTHESIZED_WIRE_18);


b2v_mux0 : mux0
PORT MAP(sel => SYNTHESIZED_WIRE_5,
		 a => SYNTHESIZED_WIRE_18,
		 b => SYNTHESIZED_WIRE_19,
		 y => SYNTHESIZED_WIRE_12);


b2v_mux1 : mux1
PORT MAP(sel => SYNTHESIZED_WIRE_8,
		 a => SYNTHESIZED_WIRE_9,
		 b => SYNTHESIZED_WIRE_10,
		 y => SYNTHESIZED_WIRE_2);


b2v_program_counter : pc
PORT MAP(clk => clk,
		 current_instr => SYNTHESIZED_WIRE_17,
		 next_instr => SYNTHESIZED_WIRE_17);


b2v_registers_file : registers
PORT MAP(clk => clk,
		 rd_addr => SYNTHESIZED_WIRE_12,
		 rs_addr => SYNTHESIZED_WIRE_13,
		 rt_addr => SYNTHESIZED_WIRE_18,
		 wr_data => SYNTHESIZED_WIRE_15,
		 rs => SYNTHESIZED_WIRE_1,
		 rt => SYNTHESIZED_WIRE_9);


b2v_sign_extend : sign_extend
PORT MAP(data_in => SYNTHESIZED_WIRE_19,
		 data_out => SYNTHESIZED_WIRE_10);


END bdf_type;
