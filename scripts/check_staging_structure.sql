-- Script para verificar a estrutura real da tabela INTEGRACAOPRODUTOSTAGING

-- 1. Descrever a estrutura da tabela
DESC INTEGRACAOPRODUTOSTAGING;

-- 2. Ver alguns registros exemplo
select *
  from integracaoprodutostaging
 where rownum <= 3;

-- 3. Ver os nomes das colunas
select column_name,
       data_type,
       data_length,
       nullable
  from user_tab_columns
 where table_name = 'INTEGRACAOPRODUTOSTAGING'
 order by column_id;