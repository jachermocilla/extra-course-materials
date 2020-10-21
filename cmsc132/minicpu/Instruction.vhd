library ieee;
use ieee.std_logic_1164.all;
use ieee.numeric_std.all;

entity Instruction is
  port(
    instr_addr : in std_logic_vector(2 downto 0) := "000";    -- instruction address

    op         : out std_logic_vector(1 downto 0);   -- operation code
    rs_addr    : out std_logic_vector(1 downto 0);   -- source register 1 address
    rt_addr    : out std_logic_vector(1 downto 0);   -- source register 2 address
    rd_addr    : out std_logic_vector(1 downto 0)   -- destination register address
  );
end Instruction;


architecture behavioral of Instruction is

  ------- Program -------
  -- Buy pencil    2 Baht
  -- Buy eraser    1 Baht
  -- Pay           3 Baht
  -- Change        0 Baht
  -----------------------

  ------------ Python -------------
  -- pencil = 2
  -- eraser = 1
  -- pay = 3
  -- change = pay - (pencil + eraser)
  ---------------------------------

  type instruction_set is array(0 to 7) of std_logic_vector(7 downto 0);
  constant instr : instruction_set := (
    "11000010",  -- addi $s0, $s0, 2
    "11010101",  -- addi $s1, $s1, 1
    "11101011",  -- addi $s2, $s2, 3
    "01000111",  -- add  $s3, $s0, $s1
    "10101100",  -- sub  $s0, $s2, $s3
    "00000000",
    "00000000",
    "00000000"
  );

begin

  op <= instr(to_integer(unsigned(instr_addr)))(7 downto 6);
  rs_addr <= instr(to_integer(unsigned(instr_addr)))(5 downto 4);
  rt_addr <= instr(to_integer(unsigned(instr_addr)))(3 downto 2);
  rd_addr <= instr(to_integer(unsigned(instr_addr)))(1 downto 0);

end behavioral;
