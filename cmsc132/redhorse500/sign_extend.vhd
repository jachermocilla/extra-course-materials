library ieee;
use ieee.std_logic_1164.all;

entity sign_extend is
  port(
    data_in  : in std_logic_vector(1 downto 0);
    data_out : out std_logic_vector(7 downto 0)
  );
end sign_extend;


architecture dataflow of sign_extend is
begin
  data_out <= "000000" & data_in;
end dataflow;