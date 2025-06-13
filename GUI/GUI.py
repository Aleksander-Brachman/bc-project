import tkinter as tk
from tkinter import ttk, messagebox, simpledialog
import mysql.connector

# ---- DANE DOSTĘPU DO BAZY ----
DB_HOST = "localhost"
DB_USER = "root"
DB_PASS = "123"   # <- tutaj wpisz swoje hasło do root
DB_NAME = "project_db"

USERS = ["user_1", "user_2", "user_3"]

class AnnouncementApp:
    def __init__(self, root):
        self.root = root
        self.root.title("Announcement GUI")
        self.user = tk.StringVar(value=USERS[0])

        self.connect_db()

        self.show_login()

    def connect_db(self):
        try:
            self.conn = mysql.connector.connect(
                host=DB_HOST,
                user=DB_USER,
                password=DB_PASS,
                database=DB_NAME
            )
            self.cursor = self.conn.cursor(dictionary=True)
        except Exception as e:
            messagebox.showerror("Błąd połączenia z bazą", str(e))
            self.root.quit()

    def show_login(self):
        for widget in self.root.winfo_children():
            widget.destroy()
        tk.Label(self.root, text="Wybierz użytkownika:", font=("Arial", 14)).pack(pady=10)
        user_menu = ttk.Combobox(self.root, values=USERS, textvariable=self.user, state="readonly")
        user_menu.pack(pady=5)
        tk.Button(self.root, text="Dalej", command=self.show_main).pack(pady=10)

    def show_main(self):
        for widget in self.root.winfo_children():
            widget.destroy()
        tk.Label(self.root, text=f"Zalogowano jako: {self.user.get()}", font=("Arial", 12)).pack(pady=5)

        # Tabela
        self.tree = ttk.Treeview(self.root, columns=("id", "author", "date", "msg"), show="headings", height=10)
        for col in ("id", "author", "date", "msg"):
            self.tree.heading(col, text=col)
            self.tree.column(col, minwidth=0, width=120, stretch=True)
        self.tree.pack(pady=5)
        self.load_table()

        # Dodawanie wpisu
        add_frame = tk.Frame(self.root)
        add_frame.pack(pady=5)
        tk.Label(add_frame, text="Nowa wiadomość:").grid(row=0, column=0)
        self.new_msg_entry = tk.Entry(add_frame, width=40)
        self.new_msg_entry.grid(row=0, column=1, padx=5)
        tk.Button(add_frame, text="Wyślij", command=self.add_announcement).grid(row=0, column=2, padx=5)

        # Aktualizacja wpisu
        upd_frame = tk.Frame(self.root)
        upd_frame.pack(pady=5)
        tk.Label(upd_frame, text="ID do aktualizacji:").grid(row=0, column=0)
        self.update_id_entry = tk.Entry(upd_frame, width=5)
        self.update_id_entry.grid(row=0, column=1, padx=5)
        tk.Label(upd_frame, text="Nowa wiadomość:").grid(row=0, column=2)
        self.update_msg_entry = tk.Entry(upd_frame, width=30)
        self.update_msg_entry.grid(row=0, column=3, padx=5)
        tk.Button(upd_frame, text="Aktualizuj", command=self.update_announcement).grid(row=0, column=4, padx=5)

        # Odświeżanie tabeli
        tk.Button(self.root, text="Odśwież", command=self.load_table).pack(pady=5)

        # Wyloguj
        tk.Button(self.root, text="Wyloguj", command=self.show_login).pack(pady=5)

    def load_table(self):
        # Wyczysc tabelę
        for i in self.tree.get_children():
            self.tree.delete(i)
        self.cursor.execute("SELECT * FROM announcement ORDER BY id")
        for row in self.cursor.fetchall():
            self.tree.insert("", "end", values=(row["id"], row["author"], row["date"], row["msg"]))

    def add_announcement(self):
        msg = self.new_msg_entry.get()
        author = self.user.get()
        if not msg:
            messagebox.showerror("Błąd", "Wpisz wiadomość!")
            return
        try:
            sql = "INSERT INTO announcement (msg, author) VALUES (%s, %s)"
            self.cursor.execute(sql, (msg, author))
            self.conn.commit()
            self.load_table()
            self.new_msg_entry.delete(0, tk.END)
        except Exception as e:
            messagebox.showerror("Błąd", str(e))

    def update_announcement(self):
        try:
            id_ = int(self.update_id_entry.get())
            msg = self.update_msg_entry.get()
            author = self.user.get()
            if not msg:
                messagebox.showerror("Błąd", "Wpisz wiadomość!")
                return
            sql = "UPDATE announcement SET msg=%s, author=%s, date=NOW() WHERE id=%s"
            self.cursor.execute(sql, (msg, author, id_))
            if self.cursor.rowcount == 0:
                messagebox.showerror("Błąd", "Nie ma wpisu o takim ID!")
                return
            self.conn.commit()
            self.load_table()
            self.update_id_entry.delete(0, tk.END)
            self.update_msg_entry.delete(0, tk.END)
        except ValueError:
            messagebox.showerror("Błąd", "Podaj poprawne ID (liczba)!")
        except Exception as e:
            messagebox.showerror("Błąd", str(e))

if __name__ == "__main__":
    root = tk.Tk()
    app = AnnouncementApp(root)
    root.mainloop()
