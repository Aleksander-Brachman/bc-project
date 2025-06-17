# bc-project

**Temat projektu:** Ochrona integralności komunikatów z wykorzystaniem Hyperledger Fabric

**Autorzy projektu:** Aleksander Brachman, Krzysztof Skociński

Celem projektu w ramach przedmiotu "Technologia Blockchain" było utworzenie systemu przechowywania komunikatów (wiadomości), które są zabezpieczane przed nieautoryzowanymi modyfikacjami dzięki integracji systemu z Hyperledger Fabric, gdzie w sieci blockchainowej umieszczane są kopie komunikatów z bazy danych. 


**Wykorzystane technologie w projekcie:**
- System operacyjny Linux Debian
- System zarządzania bazą danych MariaDB
- Platforma blockchainowa Hyperledger Fabric
- Języki programowania: GO (smart contract oraz skrypt), Python (GUI)

**Krótki opis działania systemu:**
Za pośrednictwiem GUI, które jest częścią projektu użytkownik (zdefiniowany jako user_1, user_2 lub user_3) może:
- dodać nowy komunikat do bazy danych SQL, którego kopia (w ciągu max 5 sekund) trafia także do ledgera w HF jako nowy asset
- zaktualizować istniejący już komunikat w bazie danych SQL:
    - jeżeli aktualizacji komunikatu dokonuje **ten sam** użytkownik, który jest widoczny jako autor danego komunikatu to następuje (w ciągu max 5 sekund) aktualizacja odpowiedniego assetu w HF, aby zawsze przechowywać ostatnią prawidłową wersję danego komunikatu,
    - jeżeli aktualizacji komunikatu dokonuje **inny** użytkownik niż ten, który jest widoczny jako autor danego komunikatu to, na bazie wartości assetu z ledgera, następuje (w ciągu max 5 sekund) aktualizacja odpowiedniego wpisu w bazie danych SQL, która przywraca do bazy danych ostatnią prawidłową wersję danego komunikatu.

**Przykładowe działanie systemu:**

![image](https://github.com/user-attachments/assets/16f82ddd-8139-4fe9-89ff-9f979736679f)
